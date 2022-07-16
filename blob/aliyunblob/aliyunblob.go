package aliyunblob

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/driver"
	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/internal/escape"
)

const (
	tokenRefreshTolerance = 300
)

type Config struct {
	Endpoint        string // OSS endpoint
	AccessKeyID     string // AccessId
	AccessKeySecret string // AccessKey
	BucketName      string // BucketName
}

type AuthProxy struct {
	Host     string
	User     string
	Password string
}

type Options struct {
	UseCname         bool
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	SecurityToken    string
	EnableMD5        bool
	EnableCRC        bool
	Proxy            string
	AuthProxy        *AuthProxy
}

const (
	defaultPageSize         = 1000
	defaultConnectTimeout   = 30 * time.Second
	defaultReadWriteTimeout = 60 * time.Second
)

func init() {
	blob.DefaultURLMux().RegisterBucket(Scheme, &URLOpener{})
}

// Scheme is the URL scheme aliyunblob registers its URLOpener under on
// blob.DefaultMux.
const Scheme = "aliyun"

// URLOpener opens Azure URLs like "aliyun://my-bucket".
type URLOpener struct{}

// OpenBucketURL opens a blob.Bucket based on u.
func (o *URLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	cfg := &Config{}
	q := u.Query()

	cfg.Endpoint = q.Get("endpoint")
	cfg.AccessKeyID = q.Get("accessKeyId")
	cfg.AccessKeySecret = q.Get("accessKeySecret")
	cfg.BucketName = u.Host

	opts := new(Options)
	err := setOptionsFromURLParams(q, opts)
	if err != nil {
		return nil, err
	}
	return OpenBucket(ctx, cfg, opts)
}

func setOptionsFromURLParams(q url.Values, o *Options) error {
	for param, values := range q {
		value := values[0]
		switch param {
		case "useCname":
			o.UseCname = value == "true"
		case "connectTimeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			o.ConnectTimeout = d
		case "readWriteTimeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			o.ReadWriteTimeout = d
		case "securityToken":
			o.SecurityToken = value
		case "enableMD5":
			o.EnableMD5 = value == "true"
		case "enableCRC":
			o.EnableCRC = value == "true"
		case "proxy":
			o.Proxy = value
		case "authProxyHost":
			o.AuthProxy.Host = value
		case "authProxyUser":
			o.AuthProxy.User = value
		case "authProxyPassword":
			o.AuthProxy.Password = value
		}
	}

	return nil
}

// OpenBucket returns a *blob.Bucket.
func OpenBucket(ctx context.Context, cfg *Config, opts *Options) (*blob.Bucket, error) {
	b, err := openBucket(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}
	return blob.NewBucket(b), nil
}

func openBucket(ctx context.Context, cfg *Config, opts *Options) (*bucket, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("aliyunblob.OpenBucket: endpoint is required")
	}

	if cfg.AccessKeyID == "" {
		return nil, errors.New("aliyunblob.OpenBucket accessKeyID is required")
	}

	if cfg.AccessKeySecret == "" {
		return nil, errors.New("aliyunblob.OpenBucket accessKeySecret is required")
	}

	if cfg.BucketName == "" {
		return nil, errors.New("aliyunblob.OpenBucket bucketName is required")
	}

	options := []oss.ClientOption{}

	if opts == nil {
		opts = &Options{}
	}
	if opts.UseCname {
		options = append(options, oss.UseCname(true))
	}

	connectTimeout := defaultConnectTimeout
	readWriteTimeout := defaultReadWriteTimeout

	if opts.ConnectTimeout > 0 {
		connectTimeout = opts.ConnectTimeout
	}
	if opts.ReadWriteTimeout > 0 {
		readWriteTimeout = opts.ReadWriteTimeout
	}
	options = append(options, oss.Timeout(int64(connectTimeout.Seconds()), int64(readWriteTimeout.Seconds())))

	if opts.SecurityToken != "" {
		options = append(options, oss.SecurityToken(opts.SecurityToken))
	}

	if opts.EnableMD5 {
		options = append(options, oss.EnableMD5(true))
	}

	if opts.EnableCRC {
		options = append(options, oss.EnableCRC(true))
	}

	if opts.Proxy != "" {
		options = append(options, oss.Proxy(opts.Proxy))
	}

	if opts.AuthProxy != nil {
		options = append(options, oss.AuthProxy(opts.AuthProxy.Host, opts.AuthProxy.User, opts.AuthProxy.Password))
	}

	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, options...)

	if err != nil {
		return nil, err
	}

	ob, err := client.Bucket(cfg.BucketName)

	if err != nil {
		return nil, err
	}

	return &bucket{ob: ob}, nil
}

type bucket struct {
	ob *oss.Bucket
}

// ErrorCode should return a code that describes the error, which was returned by
// one of the other methods in this interface.
func (b *bucket) ErrorCode(err error) gdkerr.ErrorCode {
	e, ok := err.(*oss.ServiceError)
	if !ok {
		return gdkerr.Unknown
	}
	switch e.Code {
	case "NoSuchBucket", "NoSuchKey":
		return gdkerr.NotFound
	case "AccessDenied":
		return gdkerr.PermissionDenied
	case "InvalidArgument":
		return gdkerr.InvalidArgument
	default:
		return gdkerr.Unknown
	}
}

// As converts i to driver-specific types.
// See https://gocloud.dev/concepts/as/ for background information.
func (b *bucket) As(i interface{}) bool {
	p, ok := i.(**oss.Bucket)
	if !ok {
		return false
	}
	*p = b.ob
	return true
}

// ErrorAs allows drivers to expose driver-specific types for returned
// errors.
// See https://gocloud.dev/concepts/as/ for background information.
func (b *bucket) ErrorAs(err error, i interface{}) bool {
	return errors.As(err, i)
}

// Attributes returns attributes for the blob. If the specified object does
// not exist, Attributes must return an error for which ErrorCode returns
// gdkerr.NotFound.
// The portable type will not modify the returned Attributes.
func (b *bucket) Attributes(ctx context.Context, key string) (*driver.Attributes, error) {
	key = escapeKey(key)

	resp, err := b.ob.GetObjectDetailedMeta(key)

	if err != nil {
		return nil, err
	}

	modTime, err := http.ParseTime(resp.Get("Last-Modified"))

	if err != nil {
		return nil, err
	}

	md := make(map[string]string)

	for k, v := range resp {
		if strings.HasPrefix(k, "x-oss-meta-") {
			md[k[len("x-oss-meta-"):]] = strings.Join(v, ",")
		}
	}

	size, err := strconv.ParseInt(resp.Get("Content-Length"), 10, 64)

	if err != nil {
		return nil, err
	}

	eTag := resp.Get("ETag")
	md5 := eTagToMD5(&eTag)

	return &driver.Attributes{
		CacheControl:       resp.Get("Cache-Control"),
		ContentDisposition: resp.Get("Content-Disposition"),
		ContentEncoding:    resp.Get("Content-Encoding"),
		ContentLanguage:    resp.Get("Content-Language"),
		ContentType:        resp.Get("Content-Type"),
		Metadata:           md,
		// CreateTime not supported; left as the zero time.
		ModTime: modTime,
		Size:    size,
		MD5:     md5,
		ETag:    eTag,
		AsFunc: func(i interface{}) bool {
			p, ok := i.(*http.Header)
			if !ok {
				return false
			}
			*p = resp
			return true
		},
	}, nil

}

// ListPaged lists objects in the bucket, in lexicographical order by
// UTF-8-encoded key, returning pages of objects at a time.
// Services are only required to be eventually consistent with respect
// to recently written or deleted objects. That is to say, there is no
// guarantee that an object that's been written will immediately be returned
// from ListPaged.
// opts is guaranteed to be non-nil.
func (b *bucket) ListPaged(ctx context.Context, opts *driver.ListOptions) (*driver.ListPage, error) {
	in := []oss.Option{}

	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}

	in = append(in, oss.MaxKeys(pageSize))

	if len(opts.PageToken) > 0 {
		in = append(in, oss.ContinuationToken(string(opts.PageToken)))
	}

	if opts.Prefix != "" {
		in = append(in, oss.Prefix(opts.Prefix))
	}

	if opts.Delimiter != "" {
		in = append(in, oss.Delimiter(opts.Delimiter))
	}

	resp, err := b.ob.ListObjectsV2(in...)

	if err != nil {
		return nil, err
	}

	page := driver.ListPage{}

	if resp.NextContinuationToken != "" {
		page.NextPageToken = []byte(resp.NextContinuationToken)
	}

	if n := len(resp.Objects) + len(resp.CommonPrefixes); n > 0 {
		page.Objects = make([]*driver.ListObject, n)
		for i, obj := range resp.Objects {
			obj := obj
			page.Objects[i] = &driver.ListObject{
				Key:     unescapeKey(obj.Key),
				ModTime: obj.LastModified,
				Size:    obj.Size,
				MD5:     eTagToMD5(&obj.ETag),
				AsFunc: func(i interface{}) bool {
					p, ok := i.(*oss.ObjectProperties)
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
			page.Objects[i+len(resp.Objects)] = &driver.ListObject{
				Key:   unescapeKey(prefix),
				IsDir: true,
				AsFunc: func(i interface{}) bool {
					p, ok := i.(*string)
					if !ok {
						return false
					}
					*p = prefix
					return true
				},
			}
		}
		if len(page.Objects) > 0 && len(resp.CommonPrefixes) > 0 {
			// S3 gives us blobs and "directories" in separate lists; sort them.
			sort.Slice(page.Objects, func(i, j int) bool {
				return page.Objects[i].Key < page.Objects[j].Key
			})
		}
	}

	return &page, nil
}

type reader struct {
	io.ReadCloser
	attrs *driver.ReaderAttributes
}

func (r *reader) As(i interface{}) bool {

	return true
}

func (r *reader) Attributes() *driver.ReaderAttributes {
	return r.attrs
}

// NewRangeReader returns a Reader that reads part of an object, reading at
// most length bytes starting at the given offset. If length is negative, it
// will read until the end of the object. If the specified object does not
// exist, NewRangeReader must return an error for which ErrorCode returns
// gdkerr.NotFound.
// opts is guaranteed to be non-nil.
func (b *bucket) NewRangeReader(ctx context.Context, key string, offset int64, length int64, opts *driver.ReaderOptions) (driver.Reader, error) {
	key = escapeKey(key)

	in := []oss.Option{}

	attrs, err := b.Attributes(ctx, key)
	if err != nil {
		return nil, err
	}

	r, err := b.ob.GetObject(key, in...)

	if err != nil {
		return nil, err
	}

	return &reader{ReadCloser: r, attrs: &driver.ReaderAttributes{
		ContentType: attrs.ContentType,
		ModTime:     attrs.ModTime,
		Size:        attrs.Size,
	}}, nil

}

var _ driver.Writer = (*writer)(nil)

// writer writes an S3 object, it implements io.WriteCloser.
type writer struct {
	w     *io.PipeWriter
	ctx   context.Context
	donec chan struct{} // closed when done writing
	// The following fields will be written before donec closes:
	err error

	ob  *oss.Bucket
	key string
	in  []oss.Option
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

		err = w.ob.PutObject(w.key, body, w.in...)

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

// NewTypedWriter returns Writer that writes to an object associated with key.
//
// A new object will be created unless an object with this key already exists.
// Otherwise any previous object with the same key will be replaced.
// The object may not be available (and any previous object will remain)
// until Close has been called.
//
// contentType sets the MIME type of the object to be written. It must not be
// empty. opts is guaranteed to be non-nil.
//
// The caller must call Close on the returned Writer when done writing.
//
// Implementations should abort an ongoing write if ctx is later canceled,
// and do any necessary cleanup in Close. Close should then return ctx.Err().
func (b *bucket) NewTypedWriter(ctx context.Context, key string, contentType string, opts *driver.WriterOptions) (driver.Writer, error) {
	key = escapeKey(key)

	in := []oss.Option{
		oss.ContentType(contentType),
	}

	if opts.CacheControl != "" {
		in = append(in, oss.CacheControl(opts.CacheControl))
	}

	if opts.ContentEncoding != "" {
		in = append(in, oss.ContentEncoding(opts.ContentEncoding))
	}

	if opts.ContentDisposition != "" {
		in = append(in, oss.ContentDisposition(opts.ContentDisposition))
	}

	if opts.ContentLanguage != "" {
		in = append(in, oss.ContentLanguage(opts.ContentLanguage))
	}

	if len(opts.ContentMD5) > 0 {
		in = append(in, oss.ContentMD5(base64.StdEncoding.EncodeToString(opts.ContentMD5)))
	}

	if opts.BeforeWrite != nil {
		if err := opts.BeforeWrite(func(interface{}) bool { return false }); err != nil {
			return nil, err
		}
	}

	return &writer{
		ctx:   ctx,
		ob:    b.ob,
		key:   key,
		in:    in,
		donec: make(chan struct{}),
	}, nil
}

// Copy copies the object associated with srcKey to dstKey.
//
// If the source object does not exist, Copy must return an error for which
// ErrorCode returns gdkerr.NotFound.
//
// If the destination object already exists, it should be overwritten.
//
// opts is guaranteed to be non-nil.
func (b *bucket) Copy(ctx context.Context, dstKey string, srcKey string, opts *driver.CopyOptions) error {
	srcKey = escapeKey(srcKey)
	dstKey = escapeKey(dstKey)

	in := []oss.Option{}

	if opts.BeforeCopy != nil {
		return opts.BeforeCopy(func(interface{}) bool { return false })
	}

	_, err := b.ob.CopyObject(srcKey, dstKey, in...)

	return err
}

// Delete deletes the object associated with key. If the specified object does
// not exist, Delete must return an error for which ErrorCode returns
// gdkerr.NotFound.
func (b *bucket) Delete(ctx context.Context, key string) error {
	key = escapeKey(key)
	return b.ob.DeleteObject(key)
}

// SignedURL returns a URL that can be used to GET the blob for the duration
// specified in opts.Expiry. opts is guaranteed to be non-nil.
// If not supported, return an error for which ErrorCode returns
// gdkerr.Unimplemented.
func (b *bucket) SignedURL(ctx context.Context, key string, opts *driver.SignedURLOptions) (string, error) {
	key = escapeKey(key)

	var expiredInSec int64 = 60

	if opts.Expiry != 0 {
		expiredInSec = int64(opts.Expiry.Seconds())
	}

	var method oss.HTTPMethod = oss.HTTPGet
	if opts.Method != "" {
		switch opts.Method {
		case "PUT":
			method = oss.HTTPPost
		default:
			method = oss.HTTPMethod(opts.Method)
		}
	}

	in := []oss.Option{}

	switch method {
	case oss.HTTPPost:
		if opts.EnforceAbsentContentType || opts.ContentType != "" {
			in = append(in, oss.ContentType(opts.ContentType))
		}
	}

	return b.ob.SignURL(key, method, expiredInSec, in...)
}

// Close cleans up any resources used by the Bucket. Once Close is called,
// there will be no method calls to the Bucket other than As, ErrorAs, and
// ErrorCode. There may be open readers or writers that will receive calls.
// It is up to the driver as to how these will be handled.
func (b *bucket) Close() error {
	return nil
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
