package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/cloudflare/cloudflare-go"
)

func (c *App) NewMediaStorage() (*s3.Client, error) {

	endpoint := c.Config.Storage.Endpoint
	accessKeyId := c.Config.Storage.AccessKeyID
	accessKeySecret := c.Config.Storage.AccessKeySecret

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	return client, nil
}

func (c *App) GetPresignedURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := c.LoggedInUser(r)

		if c.Config.Restrictions.Media.VerifiedOnly && !user.Verified {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "You must be verified to upload media",
				},
			})
			return
		}

		query := r.URL.Query()
		filetype := query.Get("filetype")

		filename := RandomString(32)
		key := fmt.Sprintf("media/attachments/%s.%s", filename, filetype)

		presignClient := s3.NewPresignClient(c.MediaStorage)

		presignResult, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(c.Config.Storage.BucketName),
			Key:    aws.String(key),
		})

		if err != nil {
			RespondWithJSON(w, &JSONResponse{
				Code: http.StatusOK,
				JSON: map[string]any{
					"error": "couldn't get presigned URL",
				},
			})
			return
		}
		log.Println("returning ", key)

		resp := map[string]any{
			"url": presignResult.URL,
			"key": key,
		}

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: resp,
		})

	}
}

func (c *App) GetUploadURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := RandomString(32)

		api, err := cloudflare.NewWithAPIToken(c.Config.Images.APIToken)
		if err != nil {
			log.Println(err)
		}

		ctx := context.Background()

		aid := cloudflare.AccountIdentifier(c.Config.Images.AccountID)

		rsu := true

		u, err := api.CreateImageDirectUploadURL(ctx, aid, cloudflare.CreateImageDirectUploadURLParams{
			RequireSignedURLs: &rsu,
			Metadata: map[string]interface{}{
				"id": id,
			},
		})
		if err != nil {
			log.Println(err)
		}

		fmt.Println(u)

		RespondWithJSON(w, &JSONResponse{
			Code: http.StatusOK,
			JSON: u,
		})

	}
}
