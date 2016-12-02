package main

import (
	"fmt"
	"log"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	QueueUrl       = "https://sqs.ap-southeast-1.amazonaws.com/449696992066/testing"
	FromBucketName = "mediatek-sync-testing"
	ToBucketName   = "mediatek-to-testing"
	Region         = "ap-southeast-1"
)

func main() {
	sess := session.New(&aws.Config{
		Region:     aws.String(Region),
		MaxRetries: aws.Int(5),
	})

	svc := sqs.New(sess)

	receiveParams := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(QueueUrl),
		MaxNumberOfMessages: aws.Int64(3),  // 一次最多取幾個 message
		VisibilityTimeout:   aws.Int64(30), // 如果這個 message 沒刪除，下次再被取出來的時間
		WaitTimeSeconds:     aws.Int64(20), // long polling 方式取，會建立一條長連線並且等在那邊，直到 SQS 收到新 message 回傳給這條連線才中斷
	}

	receive_resp, err := svc.ReceiveMessage(receiveParams)

	if err != nil {
		log.Println(err)
	}

	for _, message := range receive_resp.Messages {
		b := *message.Body
		// fmt.Println(body) // struct of message, http://docs.aws.amazon.com/sdk-for-go/api/service/sqs/#Message

		var body Body
		err := json.Unmarshal([]byte(b), &body)

		if err != nil {
			fmt.Println("error:", err)
		}

		objectKey := body.Records[0].S3.Object.Key

	}
}

func getObject(s3Client *s3.S3, bucket string, key string) ([]byte, err) {
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
