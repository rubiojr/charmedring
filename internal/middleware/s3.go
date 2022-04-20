package middleware

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	mlog "github.com/rubiojr/charmedring/internal/log"
)

func multipartReader(r *http.Request, reader io.Reader) (*multipart.Reader, error) {
	v := r.Header.Get("Content-Type")
	if v == "" {
		return nil, http.ErrNotMultipart
	}

	d, params, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		return nil, http.ErrNotMultipart
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, http.ErrMissingBoundary
	}

	return multipart.NewReader(reader, boundary), nil
}

func fileHeader(r *http.Request, src io.Reader) (*multipart.FileHeader, error) {
	mr, err := multipartReader(r, src)
	if err != nil {
		return nil, err
	}

	form, err := mr.ReadForm(gin.Default().MaxMultipartMemory)
	if err != nil {
		return nil, err
	}
	data := form.File["data"]
	if data == nil {
		s3Errorf("missing data field")
		return nil, fmt.Errorf("missing data field")
	}

	if len(data) != 1 {
		s3Errorf("expected exactly one data field")
		return nil, fmt.Errorf("expected exactly one data field")
	}
	file := data[0]

	return file, nil
}

func S3(endpoint, accessKeyID, secretAccessKey, bucketName, location string) (gin.HandlerFunc, error) {
	s3Infof("init %s/%s %s", endpoint, bucketName, location)
	uploader, err := newUploader(endpoint, accessKeyID, secretAccessKey, bucketName, location)
	if err != nil {
		s3Errorf("could not initialized middleware: %s", err)
		return nil, err
	}

	return func(ctx *gin.Context) {
		rb := &binding{}
		err := ctx.ShouldBindBodyWith(struct{}{}, rb)
		if err != nil {
			s3Errorf("request body read failed: %s", err)
			return
		}

		c := ctx.Copy()
		go func() {
			header, err := fileHeader(c.Request, rb.buf)
			if err != nil {
				s3Errorf("invalid multipart file: %s", err)
				return
			}

			file, err := header.Open()
			if err != nil {
				s3Errorf("error opening file: %s", err)
				return
			}
			defer file.Close()

			path := c.Request.URL.Path

			cid, ok := c.Value("charm_id").(string)
			if !ok {
				c.String(http.StatusBadRequest, "charm_id not found")
				return
			}
			obj := filepath.Join(cid, strings.TrimPrefix(path, "/v1/fs/"))
			s3Debugf("uploading %s, %d bytes", obj, header.Size)
			err = uploader.upload(c, obj, file, header.Size)
			if err != nil {
				s3Errorf("failed uploading object %s: %s", obj, err)
			}
		}()
	}, nil
}

type uploader struct {
	s3         *minio.Client
	bucketName string
	region     string
}

func newUploader(endpointURL, accessKeyID, secretAccessKey, bucket, region string) (*uploader, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, err
	}

	endpoint := u.Host
	useSSL := true
	if u.Scheme == "http" {
		useSSL = false
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		s3Errorf("could not initialize the S3 client: %s", err)
		return nil, err
	}

	ctx := context.Background()
	err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			s3Debugf("bucket %s already exists", bucket)
		} else {
			s3Errorf("error creating the bucket, hoping for the best: %s", err)
		}
	} else {
		s3Debugf("bucket %s successfully created", bucket)
	}

	return &uploader{minioClient, bucket, region}, nil
}

func (u *uploader) upload(ctx context.Context, name string, reader io.Reader, len int64) error {
	_, err := u.s3.PutObject(ctx, u.bucketName, name, reader, len, minio.PutObjectOptions{})
	if err != nil {
		s3Errorf("object PUT failed: %s", err)
		return err
	}

	s3Debugf("uploaded %s", name)
	return nil
}

func s3Errorf(msg string, args ...interface{}) {
	mlog.Errorf(fmt.Sprintf("[s3] %s", msg), args...)
}

func s3Debugf(msg string, args ...interface{}) {
	mlog.Debugf(fmt.Sprintf("[s3] %s", msg), args...)
}

func s3Infof(msg string, args ...interface{}) {
	mlog.Infof(fmt.Sprintf("[s3] %s", msg), args...)
}
