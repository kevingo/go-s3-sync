package main

import (
	"bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"io"
	"log"
	"net/url"
	"os"
)

var logger = initLogger()

func initLogger() *log.Logger {
	file, err := os.OpenFile("sync.log", os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Println("fail to create log file!")
	}
	writers := []io.Writer{
		file,
		os.Stdout,
	}
	logger := log.New(io.MultiWriter(writers...), "", log.Ldate|log.Ltime|log.Lshortfile)
	return logger
}

func main() {
	sqsClient := sqsClient()
	client := s3Client(Region)

	for {
		log.Println("Start polling message ... ")
		receive_resp, err := receiveMessage(sqsClient)
		logErr(err)

		for _, message := range receive_resp.Messages {
			b := *message.Body

			var body Body
			err1 := json.Unmarshal([]byte(b), &body)
			if err1 != nil {
				logErr(err1)
				continue
			}

			key := body.Records[0].S3.Object.Key
			objectKey, _ := url.QueryUnescape(key)

			logger.Println("Start process " + objectKey)

			file, err2 := getObject(client, FromBucketName, objectKey)
			if err2 != nil {
				logErr(err2)
				continue
			}

			logger.Println("Get " + objectKey + " success")

			result, err3 := s3Uploader(objectKey, file)
			if err3 != nil {
				logErr(err3)
				continue
			}

			if err1 == nil && err2 == nil && err3 == nil {
				deleteMessage(sqsClient, message)
				logger.Println("Success upload file ", result.Location)
			}
		}
	}
}

func receiveMessage(svc *sqs.SQS) (*sqs.ReceiveMessageOutput, error) {
	receiveParams := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(QueueUrl),
		MaxNumberOfMessages: aws.Int64(5),
		VisibilityTimeout:   aws.Int64(3600),
		WaitTimeSeconds:     aws.Int64(20),
	}

	receive_resp, err := svc.ReceiveMessage(receiveParams)
	return receive_resp, err
}

func deleteMessage(svc *sqs.SQS, message *sqs.Message) {
	delete_params := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(QueueUrl),  // Required
		ReceiptHandle: message.ReceiptHandle, // Required
	}
	_, err := svc.DeleteMessage(delete_params) // No response returned when successed.

	logErr(err)
}

func getObject(s3Client *s3.S3, bucket string, key string) ([]byte, error) {
	results, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, err
	}

	defer results.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, results.Body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func s3Uploader(key string, body []byte) (*s3manager.UploadOutput, error) {
	credentials := *credentials.NewStaticCredentials(CNAccessKey, CNSecretKey, "")
	config := aws.Config{
		Credentials: &credentials,
		Region:      aws.String(CNRegion),
	}
	uploader := s3manager.NewUploader(session.New(&config))
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   bytes.NewReader(body),
		Bucket: aws.String(ToBucketName),
		Key:    aws.String(key),
	})

	return result, err
}

func s3Client(region string) *s3.S3 {
	s3Config := &aws.Config{
		Region: aws.String(region),
	}

	newSession := session.New(s3Config)
	client := s3.New(newSession)
	return client
}

func sqsClient() *sqs.SQS {
	sess := session.New(&aws.Config{
		Region:     aws.String(Region),
		MaxRetries: aws.Int(5),
	})

	svc := sqs.New(sess)
	return svc
}

func logErr(err error) {
	if err != nil {
		logger.Println("ERROR:", err)
	}
}
