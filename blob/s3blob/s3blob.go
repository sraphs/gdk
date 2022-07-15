// Package s3blob provides a blob implementation that uses S3. Use OpenBucket
// to construct a *blob.Bucket.
//
// # URLs
//
// For blob.OpenBucket, s3blob registers for the scheme "s3".
// The default URL opener will use an AWS session with the default credentials
// and configuration; see https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
// for more details.
// To customize the URL opener, or for more details on the URL format,
// see URLOpener.
// See https://github.com/sraphs/gdk/concepts/urls/ for background information.
//
// # Escaping
//
// Go CDK supports all UTF-8 strings; to make this work with services lacking
// full UTF-8 support, strings must be escaped (during writes) and unescaped
// (during reads). The following escapes are performed for s3blob:
//   - Blob keys: ASCII characters 0-31 are escaped to "__0x<hex>__".
//     Additionally, the "/" in "../" and the trailing "/" in "//" are escaped in
//     the same way.
//   - Metadata keys: Escaped using URL encoding, then additionally "@:=" are
//     escaped using "__0x<hex>__". These characters were determined by
//     experimentation.
//   - Metadata values: Escaped using URL encoding.
//
// # As
//
// s3blob exposes the following types for As:
//   - Bucket: *s3.Client
//   - Error:  any error type returned by the service, notably smithy.APIError
//   - ListObject: typesv2.Object for objects, typesv2.CommonPrefix for "directories
//   - ListOptions.BeforeList:  *s3.ListObjectsV2Input, or *s3.ListObjectsInput
//     when Options.UseLegacyList == true
//   - Reader: s3.GetObjectInput
//   - ReaderOptions.BeforeRead: *s3.GetObjectInput
//   - Attributes: s3.HeadObjectOutput
//   - CopyOptions.BeforeCopy: s3.CopyObjectInput
//   - WriterOptions.BeforeWrite: *s3.PutObjectInput, *s3manager.Uploader
//   - SignedURLOptions.BeforeSign:
//     *s3.GetObjectInput, when Options.Method == http.MethodGet, or
//     *s3.PutObjectInput, when Options.Method == http.MethodPut, or
//     [not supported] when Options.Method == http.MethodDelete
package s3blob

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/google/wire"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/driver"
	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/internal/escape"
)

const defaultPageSize = 1000

func init() {
	blob.DefaultURLMux().RegisterBucket(Scheme, new(urlSessionOpener))
}

// Set holds Wire providers for this package.
var Set = wire.NewSet(
	wire.Struct(new(URLOpener), "ConfigProvider"),
)

type urlSessionOpener struct{}

func (o *urlSessionOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	opener := &URLOpener{}
	return opener.OpenBucketURL(ctx, u)
}

// Scheme is the URL scheme s3blob registers its URLOpener under on
// blob.DefaultMux.
const Scheme = "s3"

// URLOpener opens S3 URLs like "s3://my-bucket".
//
// The URL host is used as the bucket name.
type URLOpener struct {
	// Options specifies the options to pass to OpenBucket.
	Options Options
}

// OpenBucketURL opens a blob.Bucket based on u.
func (o *URLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	cfg, err := configFromURLParams(ctx, u.Query())
	if err != nil {
		return nil, fmt.Errorf("open bucket %v: %v", u, err)
	}
	client := s3.NewFromConfig(cfg)
	return OpenBucket(ctx, client, u.Host, &o.Options)
}

func configFromURLParams(ctx context.Context, q url.Values) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error
	for param, values := range q {
		value := values[0]
		switch param {
		case "region":
			opts = append(opts, config.WithRegion(value))
		case "profile":
			opts = append(opts, config.WithSharedConfigProfile(value))
		case "awssdk":
			// ignore, should be handled before this
		default:
			return aws.Config{}, fmt.Errorf("unknown query parameter %q", param)
		}
	}
	return config.LoadDefaultConfig(ctx, opts...)
}

// Options sets options for constructing a *blob.Bucket backed by fileblob.
type Options struct {
	// UseLegacyList forces the use of ListObjects instead of ListObjects.
	// Some S3-compatible services (like CEPH) do not currently support
	// ListObjects.
	UseLegacyList bool
}

// openBucket returns an S3 Bucket.
func openBucket(ctx context.Context, client *s3.Client, bucketName string, opts *Options) (*bucket, error) {
	if bucketName == "" {
		return nil, errors.New("s3blob.OpenBucket: bucketName is required")
	}
	if opts == nil {
		opts = &Options{}
	}

	if client == nil {
		return nil, errors.New("s3blob.OpenBucket: client is required")
	}

	return &bucket{
		name:          bucketName,
		client:        client,
		useLegacyList: opts.UseLegacyList,
	}, nil
}

// OpenBucket returns a *blob.Bucket backed by S3, using AWS SDK v2.
func OpenBucket(ctx context.Context, client *s3.Client, bucketName string, opts *Options) (*blob.Bucket, error) {
	drv, err := openBucket(ctx, client, bucketName, opts)
	if err != nil {
		return nil, err
	}
	return blob.NewBucket(drv), nil
}

// reader reads an S3 object. It implements io.ReadCloser.
type reader struct {
	body  io.ReadCloser
	attrs driver.ReaderAttributes
	raw   *s3.GetObjectOutput
}

func (r *reader) Read(p []byte) (int, error) {
	return r.body.Read(p)
}

// Close closes the reader itself. It must be called when done reading.
func (r *reader) Close() error {
	return r.body.Close()
}

func (r *reader) As(i interface{}) bool {
	p, ok := i.(*s3.GetObjectOutput)
	if !ok {
		return false
	}
	*p = *r.raw
	return true
}

func (r *reader) Attributes() *driver.ReaderAttributes {
	return &r.attrs
}

// writer writes an S3 object, it implements io.WriteCloser.
type writer struct {
	w *io.PipeWriter // created when the first byte is written

	ctx context.Context
	// v2
	uploader *manager.Uploader
	req      *s3.PutObjectInput

	donec chan struct{} // closed when done writing
	// The following fields will be written before donec closes:
	err error
}

// Write appends p to w. User must call Close to close the w after done writing.
func (w *writer) Write(p []byte) (int, error) {
	// Avoid opening the pipe for a zero-length write;
	// the concrete can do these for empty blobs.
	if len(p) == 0 {
		return 0, nil
	}
	if w.w == nil {
		// We'll write into pw and use pr as an io.Reader for the
		// Upload call to S3.
		pr, pw := io.Pipe()
		w.w = pw
		if err := w.open(pr); err != nil {
			return 0, err
		}
	}
	select {
	case <-w.donec:
		return 0, w.err
	default:
	}
	return w.w.Write(p)
}

// pr may be nil if we're Closing and no data was written.
func (w *writer) open(pr *io.PipeReader) error {

	go func() {
		defer close(w.donec)

		body := io.Reader(pr)
		if pr == nil {
			// AWS doesn't like a nil Body.
			body = http.NoBody
		}
		var err error
		w.req.Body = body
		_, err = w.uploader.Upload(w.ctx, w.req)
		if err != nil {
			w.err = err
			if pr != nil {
				pr.CloseWithError(err)
			}
			return
		}
	}()
	return nil
}

// Close completes the writer and closes it. Any error occurring during write
// will be returned. If a writer is closed before any Write is called, Close
// will create an empty file at the given key.
func (w *writer) Close() error {
	if w.w == nil {
		// We never got any bytes written. We'll write an http.NoBody.
		w.open(nil)
	} else if err := w.w.Close(); err != nil {
		return err
	}
	<-w.donec
	return w.err
}

// bucket represents an S3 bucket and handles read, write and delete operations.
type bucket struct {
	name          string
	client        *s3.Client
	useLegacyList bool
}

func (b *bucket) Close() error {
	return nil
}

func (b *bucket) ErrorCode(err error) gdkerr.ErrorCode {
	var code string
	var ae smithy.APIError
	var oe *smithy.OperationError
	if errors.As(err, &oe) && strings.Contains(oe.Error(), "301") {
		// V2 returns an OperationError with a missing redirect for invalid buckets.
		code = "NoSuchBucket"
	} else if errors.As(err, &ae) {
		code = ae.ErrorCode()
	} else {
		return gdkerr.Unknown
	}
	switch {
	case code == "NoSuchBucket" || code == "NoSuchKey" || code == "NotFound":
		return gdkerr.NotFound
	default:
		return gdkerr.Unknown
	}
}

// ListPaged implements driver.ListPaged.
func (b *bucket) ListPaged(ctx context.Context, opts *driver.ListOptions) (*driver.ListPage, error) {
	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	in := &s3.ListObjectsV2Input{
		Bucket:  aws.String(b.name),
		MaxKeys: int32(pageSize),
	}
	if len(opts.PageToken) > 0 {
		in.ContinuationToken = aws.String(string(opts.PageToken))
	}
	if opts.Prefix != "" {
		in.Prefix = aws.String(escapeKey(opts.Prefix))
	}
	if opts.Delimiter != "" {
		in.Delimiter = aws.String(escapeKey(opts.Delimiter))
	}
	resp, err := b.listObjects(ctx, in, opts)
	if err != nil {
		return nil, err
	}
	page := driver.ListPage{}
	if resp.NextContinuationToken != nil {
		page.NextPageToken = []byte(*resp.NextContinuationToken)
	}
	if n := len(resp.Contents) + len(resp.CommonPrefixes); n > 0 {
		page.Objects = make([]*driver.ListObject, n)
		for i, obj := range resp.Contents {
			obj := obj
			page.Objects[i] = &driver.ListObject{
				Key:     unescapeKey(aws.ToString(obj.Key)),
				ModTime: *obj.LastModified,
				Size:    obj.Size,
				MD5:     eTagToMD5(obj.ETag),
				AsFunc: func(i interface{}) bool {
					p, ok := i.(*types.Object)
					if !ok {
						return false
					}
					*p = obj
					return true
				},
			}
		}
		for i, prefix := range resp.CommonPrefixes {
			prefix := prefix
			page.Objects[i+len(resp.Contents)] = &driver.ListObject{
				Key:   unescapeKey(aws.ToString(prefix.Prefix)),
				IsDir: true,
				AsFunc: func(i interface{}) bool {
					p, ok := i.(*types.CommonPrefix)
					if !ok {
						return false
					}
					*p = prefix
					return true
				},
			}
		}
		if len(resp.Contents) > 0 && len(resp.CommonPrefixes) > 0 {
			// S3 gives us blobs and "directories" in separate lists; sort them.
			sort.Slice(page.Objects, func(i, j int) bool {
				return page.Objects[i].Key < page.Objects[j].Key
			})
		}
	}
	return &page, nil
}

func (b *bucket) listObjects(ctx context.Context, in *s3.ListObjectsV2Input, opts *driver.ListOptions) (*s3.ListObjectsV2Output, error) {
	if !b.useLegacyList {
		if opts.BeforeList != nil {
			asFunc := func(i interface{}) bool {
				p, ok := i.(**s3.ListObjectsV2Input)
				if !ok {
					return false
				}
				*p = in
				return true
			}
			if err := opts.BeforeList(asFunc); err != nil {
				return nil, err
			}
		}
		return b.client.ListObjectsV2(ctx, in)
	}

	// Use the legacy ListObjects request.
	legacyIn := &s3.ListObjectsInput{
		Bucket:       in.Bucket,
		Delimiter:    in.Delimiter,
		EncodingType: in.EncodingType,
		Marker:       in.ContinuationToken,
		MaxKeys:      in.MaxKeys,
		Prefix:       in.Prefix,
		RequestPayer: in.RequestPayer,
	}
	if opts.BeforeList != nil {
		asFunc := func(i interface{}) bool {
			p, ok := i.(**s3.ListObjectsInput)
			if !ok {
				return false
			}
			*p = legacyIn
			return true
		}
		if err := opts.BeforeList(asFunc); err != nil {
			return nil, err
		}
	}
	legacyResp, err := b.client.ListObjects(ctx, legacyIn)
	if err != nil {
		return nil, err
	}

	var nextContinuationToken *string
	if legacyResp.NextMarker != nil {
		nextContinuationToken = legacyResp.NextMarker
	} else if legacyResp.IsTruncated {
		nextContinuationToken = aws.String(aws.ToString(legacyResp.Contents[len(legacyResp.Contents)-1].Key))
	}
	return &s3.ListObjectsV2Output{
		CommonPrefixes:        legacyResp.CommonPrefixes,
		Contents:              legacyResp.Contents,
		NextContinuationToken: nextContinuationToken,
	}, nil
}

// As implements driver.As.
func (b *bucket) As(i interface{}) bool {
	p, ok := i.(**s3.Client)
	if !ok {
		return false
	}
	*p = b.client
	return true
}

// As implements driver.ErrorAs.
func (b *bucket) ErrorAs(err error, i interface{}) bool {
	return errors.As(err, i)
}

// Attributes implements driver.Attributes.
func (b *bucket) Attributes(ctx context.Context, key string) (*driver.Attributes, error) {
	key = escapeKey(key)
	in := &s3.HeadObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}
	resp, err := b.client.HeadObject(ctx, in)
	if err != nil {
		return nil, err
	}

	md := make(map[string]string, len(resp.Metadata))
	for k, v := range resp.Metadata {
		// See the package comments for more details on escaping of metadata
		// keys & values.
		md[escape.HexUnescape(escape.URLUnescape(k))] = escape.URLUnescape(v)
	}
	return &driver.Attributes{
		CacheControl:       aws.ToString(resp.CacheControl),
		ContentDisposition: aws.ToString(resp.ContentDisposition),
		ContentEncoding:    aws.ToString(resp.ContentEncoding),
		ContentLanguage:    aws.ToString(resp.ContentLanguage),
		ContentType:        aws.ToString(resp.ContentType),
		Metadata:           md,
		// CreateTime not supported; left as the zero time.
		ModTime: aws.ToTime(resp.LastModified),
		Size:    resp.ContentLength,
		MD5:     eTagToMD5(resp.ETag),
		ETag:    aws.ToString(resp.ETag),
		AsFunc: func(i interface{}) bool {
			p, ok := i.(*s3.HeadObjectOutput)
			if !ok {
				return false
			}
			*p = *resp
			return true
		},
	}, nil
}

// NewRangeReader implements driver.NewRangeReader.
func (b *bucket) NewRangeReader(ctx context.Context, key string, offset, length int64, opts *driver.ReaderOptions) (driver.Reader, error) {
	key = escapeKey(key)
	var byteRange *string
	if offset > 0 && length < 0 {
		byteRange = aws.String(fmt.Sprintf("bytes=%d-", offset))
	} else if length == 0 {
		// AWS doesn't support a zero-length read; we'll read 1 byte and then
		// ignore it in favor of http.NoBody below.
		byteRange = aws.String(fmt.Sprintf("bytes=%d-%d", offset, offset))
	} else if length >= 0 {
		byteRange = aws.String(fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	}
	in := &s3.GetObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
		Range:  byteRange,
	}
	if opts.BeforeRead != nil {
		asFunc := func(i interface{}) bool {
			if p, ok := i.(**s3.GetObjectInput); ok {
				*p = in
				return true
			}
			return false
		}
		if err := opts.BeforeRead(asFunc); err != nil {
			return nil, err
		}
	}
	resp, err := b.client.GetObject(ctx, in)
	if err != nil {
		return nil, err
	}
	body := resp.Body
	if length == 0 {
		body = http.NoBody
	}
	return &reader{
		body: body,
		attrs: driver.ReaderAttributes{
			ContentType: aws.ToString(resp.ContentType),
			ModTime:     aws.ToTime(resp.LastModified),
			Size:        getSize(resp.ContentLength, aws.ToString(resp.ContentRange)),
		},
		raw: resp,
	}, nil
}

// etagToMD5 processes an ETag header and returns an MD5 hash if possible.
// S3's ETag header is sometimes a quoted hexstring of the MD5. Other times,
// notably when the object was uploaded in multiple parts, it is not.
// We do the best we can.
// Some links about ETag:
// https://docs.aws.amazon.com/AmazonS3/latest/API/RESTCommonResponseHeaders.html
// https://github.com/aws/aws-sdk-net/issues/815
// https://teppen.io/2018/06/23/aws_s3_etags/
func eTagToMD5(etag *string) []byte {
	if etag == nil {
		// No header at all.
		return nil
	}
	// Strip the expected leading and trailing quotes.
	quoted := *etag
	if len(quoted) < 2 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
		return nil
	}
	unquoted := quoted[1 : len(quoted)-1]
	// Un-hex; we return nil on error. In particular, we'll get an error here
	// for multi-part uploaded blobs, whose ETag will contain a "-" and so will
	// never be a legal hex encoding.
	md5, err := hex.DecodeString(unquoted)
	if err != nil {
		return nil
	}
	return md5
}

func getSize(contentLength int64, contentRange string) int64 {
	// Default size to ContentLength, but that's incorrect for partial-length reads,
	// where ContentLength refers to the size of the returned Body, not the entire
	// size of the blob. ContentRange has the full size.
	size := contentLength
	if contentRange != "" {
		// Sample: bytes 10-14/27 (where 27 is the full size).
		parts := strings.Split(contentRange, "/")
		if len(parts) == 2 {
			if i, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				size = i
			}
		}
	}
	return size
}

// escapeKey does all required escaping for UTF-8 strings to work with S3.
func escapeKey(key string) string {
	return escape.HexEscape(key, func(r []rune, i int) bool {
		c := r[i]
		switch {
		// S3 doesn't handle these characters (determined via experimentation).
		case c < 32:
			return true
		// For "../", escape the trailing slash.
		case i > 1 && c == '/' && r[i-1] == '.' && r[i-2] == '.':
			return true
		// For "//", escape the trailing slash. Otherwise, S3 drops it.
		case i > 0 && c == '/' && r[i-1] == '/':
			return true
		}
		return false
	})
}

// unescapeKey reverses escapeKey.
func unescapeKey(key string) string {
	return escape.HexUnescape(key)
}

// NewTypedWriter implements driver.NewTypedWriter.
func (b *bucket) NewTypedWriter(ctx context.Context, key string, contentType string, opts *driver.WriterOptions) (driver.Writer, error) {
	key = escapeKey(key)
	uploader := manager.NewUploader(b.client, func(u *manager.Uploader) {
		if opts.BufferSize != 0 {
			u.PartSize = int64(opts.BufferSize)
		}
		if opts.MaxConcurrency != 0 {
			u.Concurrency = opts.MaxConcurrency
		}
	})
	md := make(map[string]string, len(opts.Metadata))
	for k, v := range opts.Metadata {
		// See the package comments for more details on escaping of metadata
		// keys & values.
		k = escape.HexEscape(url.PathEscape(k), func(runes []rune, i int) bool {
			c := runes[i]
			return c == '@' || c == ':' || c == '='
		})
		md[k] = url.PathEscape(v)
	}
	req := &s3.PutObjectInput{
		Bucket:      aws.String(b.name),
		ContentType: aws.String(contentType),
		Key:         aws.String(key),
		Metadata:    md,
	}
	if opts.CacheControl != "" {
		req.CacheControl = aws.String(opts.CacheControl)
	}
	if opts.ContentDisposition != "" {
		req.ContentDisposition = aws.String(opts.ContentDisposition)
	}
	if opts.ContentEncoding != "" {
		req.ContentEncoding = aws.String(opts.ContentEncoding)
	}
	if opts.ContentLanguage != "" {
		req.ContentLanguage = aws.String(opts.ContentLanguage)
	}
	if len(opts.ContentMD5) > 0 {
		req.ContentMD5 = aws.String(base64.StdEncoding.EncodeToString(opts.ContentMD5))
	}
	if opts.BeforeWrite != nil {
		asFunc := func(i interface{}) bool {
			pu, ok := i.(**manager.Uploader)
			if ok {
				*pu = uploader
				return true
			}
			pui, ok := i.(**s3.PutObjectInput)
			if ok {
				*pui = req
				return true
			}
			return false
		}
		if err := opts.BeforeWrite(asFunc); err != nil {
			return nil, err
		}
	}
	return &writer{
		ctx:      ctx,
		uploader: uploader,
		req:      req,
		donec:    make(chan struct{}),
	}, nil
}

// Copy implements driver.Copy.
func (b *bucket) Copy(ctx context.Context, dstKey, srcKey string, opts *driver.CopyOptions) error {
	dstKey = escapeKey(dstKey)
	srcKey = escapeKey(srcKey)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(b.name),
		CopySource: aws.String(b.name + "/" + srcKey),
		Key:        aws.String(dstKey),
	}
	if opts.BeforeCopy != nil {
		asFunc := func(i interface{}) bool {
			switch v := i.(type) {
			case **s3.CopyObjectInput:
				*v = input
				return true
			}
			return false
		}
		if err := opts.BeforeCopy(asFunc); err != nil {
			return err
		}
	}
	_, err := b.client.CopyObject(ctx, input)
	return err
}

// Delete implements driver.Delete.
func (b *bucket) Delete(ctx context.Context, key string) error {
	if _, err := b.Attributes(ctx, key); err != nil {
		return err
	}
	key = escapeKey(key)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}
	_, err := b.client.DeleteObject(ctx, input)
	return err
}

func (b *bucket) SignedURL(ctx context.Context, key string, opts *driver.SignedURLOptions) (string, error) {
	key = escapeKey(key)
	switch opts.Method {
	case http.MethodGet:
		in := &s3.GetObjectInput{
			Bucket: aws.String(b.name),
			Key:    aws.String(key),
		}
		if opts.BeforeSign != nil {
			asFunc := func(i interface{}) bool {
				v, ok := i.(**s3.GetObjectInput)
				if ok {
					*v = in
				}
				return ok
			}
			if err := opts.BeforeSign(asFunc); err != nil {
				return "", err
			}
		}
		p, err := s3.NewPresignClient(b.client, s3.WithPresignExpires(opts.Expiry)).PresignGetObject(ctx, in)
		if err != nil {
			return "", err
		}
		return p.URL, nil
	case http.MethodPut:
		in := &s3.PutObjectInput{
			Bucket: aws.String(b.name),
			Key:    aws.String(key),
		}
		if opts.EnforceAbsentContentType || opts.ContentType != "" {
			// https://github.com/aws/aws-sdk-go-v2/issues/1475
			return "", gdkerr.New(gdkerr.Unimplemented, nil, 1, "s3blob: AWS SDK v2 does not supported enforcing ContentType in SignedURLs for PUT")
		}
		if opts.BeforeSign != nil {
			asFunc := func(i interface{}) bool {
				v, ok := i.(**s3.PutObjectInput)
				if ok {
					*v = in
				}
				return ok
			}
			if err := opts.BeforeSign(asFunc); err != nil {
				return "", err
			}
		}
		p, err := s3.NewPresignClient(b.client, s3.WithPresignExpires(opts.Expiry)).PresignPutObject(ctx, in)
		if err != nil {
			return "", err
		}
		return p.URL, nil
	case http.MethodDelete:
		// https://github.com/aws/aws-sdk-java-v2/issues/2520
		return "", gdkerr.New(gdkerr.Unimplemented, nil, 1, "s3blob: AWS SDK v2 does not support SignedURL for DELETE")
	default:
		return "", fmt.Errorf("unsupported Method %q", opts.Method)
	}
}
