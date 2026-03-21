package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Service struct {
	client     *s3.S3
	bucket     string
	region     string
	baseURL    string
}

func NewS3Service(accessKey, secretKey, bucket, region string) (*S3Service, error) {
	// Create AWS session with credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	client := s3.New(sess)

	// Construct base URL for public access
	baseURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)

	logrus.WithFields(logrus.Fields{
		"bucket": bucket,
		"region": region,
		"url":    baseURL,
	}).Info("S3 service initialized")

	return &S3Service{
		client:  client,
		bucket:  bucket,
		region:  region,
		baseURL: baseURL,
	}, nil
}

// UploadImage uploads an image to S3 and returns the public URL
func (s *S3Service) UploadImage(key string, data []byte, contentType string) (string, error) {
	// Upload to S3
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to upload image to S3")
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, key)
	logrus.WithFields(logrus.Fields{
		"key": key,
		"url": publicURL,
	}).Info("Image uploaded to S3")

	return publicURL, nil
}

// UploadImageFromReader uploads an image from a reader to S3
func (s *S3Service) UploadImageFromReader(key string, reader io.Reader, contentType string) (string, error) {
	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	return s.UploadImage(key, data, contentType)
}

// DeleteImage deletes an image from S3
func (s *S3Service) DeleteImage(key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to delete image from S3")
		return fmt.Errorf("failed to delete image: %w", err)
	}

	logrus.WithField("key", key).Info("Image deleted from S3")
	return nil
}

// GetImageURL returns the public URL for an image
func (s *S3Service) GetImageURL(key string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, key)
}

// ListImages lists all images in the bucket with a prefix
func (s *S3Service) ListImages(prefix string) ([]string, error) {
	result, err := s.client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		logrus.WithError(err).WithField("prefix", prefix).Error("Failed to list images from S3")
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var urls []string
	for _, obj := range result.Contents {
		urls = append(urls, s.GetImageURL(*obj.Key))
	}

	return urls, nil
}

// UploadVideo uploads a video file to S3 and returns the public URL
func (s *S3Service) UploadVideo(key string, data []byte, contentType string) (string, error) {
	// Upload to S3
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to upload video to S3")
		return "", fmt.Errorf("failed to upload video: %w", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("%s/%s", s.baseURL, key)
	logrus.WithFields(logrus.Fields{
		"key": key,
		"url": publicURL,
	}).Info("Video uploaded to S3")

	return publicURL, nil
}

// DownloadAndUploadVideo downloads a video from a URL and uploads it to S3
func (s *S3Service) DownloadAndUploadVideo(sourceURL, s3Key string) (string, error) {
	// Download video from source
	resp, err := http.Get(sourceURL)
	if err != nil {
		logrus.WithError(err).WithField("source_url", sourceURL).Error("Failed to download video")
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	// Read video data
	videoData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read video data: %w", err)
	}

	// Upload to S3
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	return s.UploadVideo(s3Key, videoData, contentType)
}

// Close closes the S3 service (no-op for S3, but kept for interface consistency)
func (s *S3Service) Close() error {
	return nil
}

// GetPresignedURL generates a temporary public URL for a private S3 object.
// D-ID and other external services can use this to access private bucket objects.
func (s *S3Service) GetPresignedURL(key string, expiry time.Duration) (string, error) {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"key":    key,
		"expiry": expiry,
	}).Info("Generated presigned S3 URL")
	return url, nil
}
