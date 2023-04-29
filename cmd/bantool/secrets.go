package secretmanager

import (
	"context"
	"fmt"
	"log"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/golang/protobuf/proto"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func GetSecret(projectID, secretName, credentialsPath string) (string, error) {
	ctx := context.Background()

	credsOption := option.WithCredentialsFile(credentialsPath)

	client, err := secretmanager.NewClient( ctx, credsOption)
	if err != nil {
		log.fatalf("could not create secrets manager client: %v", err)
		return "", err 
	}
	defer client.Close()

	secretPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Fatalf("failed to access secret: %v", err)
		return "", err 
	}

	secretValue := proto.String(string(result.Payload.Data))
	return *secretValue, nil
}

/*
func secret() {
	projectID := os.Getenv("PROJECT_ID")
	secretName := os.Getenv("SECRET_NAME")
	credentialsPath := serviceAccOpt

	secretValue, err := secretmanager.GetSecret(projectID, secretName, credentialsPath)
	if err != nil {
		fmt.Println("Error fetching secret:", err)
	} else {
		do stuff with secret (.env etc.)
	}
}
*/