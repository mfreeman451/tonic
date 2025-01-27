package database

import (
	"context"
	"crypto/tls"
	"fmt"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/fumeapp/tonic/setting"
	opensearch "github.com/opensearch-project/opensearch-go/v2"
	signer "github.com/opensearch-project/opensearch-go/v2/signer/awsv2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
)

var Db *gorm.DB
var Os *opensearch.Client

func DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		setting.Database.Username,
		setting.Database.Password,
		setting.Database.Host,
		setting.Database.Database,
	)
}

func DURL() string {
	return "mysql://" + DSN()
}

func Setup() {

	var err error
	var logMode = logger.Error

	if setting.Database.Logging == "true" {
		logMode = logger.Info
	}

	if setting.Database.Connect == "true" {
		Db, err = gorm.Open(
			mysql.New(mysql.Config{
				DSN: DSN()},
			),
			&gorm.Config{
				Logger: logger.Default.LogMode(logMode),
			},
		)
		if err != nil {
			log.Fatalf("gorm.DB err: %v", err)
		}
	}
	if setting.Opensearch.Connect == "true" {

		if setting.Opensearch.Signed == "true" {

			awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
				awsconfig.WithRegion(setting.Aws.Region),
				awsconfig.WithCredentialsProvider(
					credentials.NewStaticCredentialsProvider(
						setting.Aws.AccessKeyID,
						setting.Aws.SecretAccessKey,
						"",
					),
				),
			)
			if err != nil {
				panic(err)
			}

			signed, err := signer.NewSignerWithService(awsCfg, "aoss")

			if err != nil {
				panic(err)
			}

			Os, err = opensearch.NewClient(opensearch.Config{Addresses: []string{setting.Opensearch.Address},
				Signer: signed,
			})

			if err != nil {
				panic(err)
			}

		} else {
			Os, err = opensearch.NewClient(opensearch.Config{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
				Addresses: []string{setting.Opensearch.Address},
				Username:  setting.Opensearch.Username,
				Password:  setting.Opensearch.Password,
			})
		}

		if err != nil {
			log.Fatalf("opensearch.NewClient err: %v", err)
		}
	}
}

func Truncate() {
	Db.Exec("DROP TABLE providers")
	Db.Exec("DROP TABLE users")
}

func Migrate() {
	Db.Exec("SQL FOR MIGRATION")
}
