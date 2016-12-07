package main

import (
	"bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"io"
	"log"
	"os"
)

var logger = initLogger()

func initLogger() *log.Logger {
	file, err := os.Create("sync.log")
	if err != nil {
		log.Fatalln("fail to create test.log file!")
	}
	logger := log.New(file, "", log.LstdFlags|log.Llongfile)

	return logger
}

func main() {
	sqsClient := sqsClient()
	client := s3Client(Region)
	// chinaClient := s3Client(CNRegion)

	for {
		receive_resp, err := receiveMessage(sqsClient)
		checkErr(err)

		for _, message := range receive_resp.Messages {
			b := *message.Body

			var body Body
			err := json.Unmarshal([]byte(b), &body)
			checkErr(err)

			objectKey := body.Records[0].S3.Object.Key
			logger.Println("Process ", objectKey)

			file, _ := getObject(client, FromBucketName, objectKey)
			result, err := s3Uploader(objectKey, file)
			checkErr(err)

			deleteMessage(sqsClient, message)
			logger.Println("Success upload file ", result.Location)
		}
	}
}

func receiveMessage(svc *sqs.SQS) (*sqs.ReceiveMessageOutput, error) {
	receiveParams := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(QueueUrl),
		MaxNumberOfMessages: aws.Int64(5),  // 一次最多取幾個 message
		VisibilityTimeout:   aws.Int64(60), // 如果這個 message 沒刪除，下次再被取出來的時間
		WaitTimeSeconds:     aws.Int64(20), // long polling 方式取，會建立一條長連線並且等在那邊，直到 SQS 收到新 message 回傳給這條連線才中斷
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

	checkErr(err)
	log.Println("[Delete message] \nMessage ID: %s has beed deleted.\n\n", *message.MessageId)
}

func getObject(s3Client *s3.S3, bucket string, key string) ([]byte, error) {
	results, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	checkErr(err)
	defer results.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, results.Body); err != nil {
		checkErr(err)
	}

	return buf.Bytes(), nil
}

func s3Uploader(key string, body []byte) (*s3manager.UploadOutput, error) {
	uploader := s3manager.NewUploader(session.New(&aws.Config{Region: aws.String(Region)}))
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

func checkErr(err error) {
	if err != nil {
		logger.Fatal("ERROR:", err)
	}
}
