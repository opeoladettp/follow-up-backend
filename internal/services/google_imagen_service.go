package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
	"github.com/sirupsen/logrus"
)

type GoogleImagenService struct {
	projectID string
	location  string
	client    *aiplatform.PredictionClient
}

func NewGoogleImagenService(projectID string) (*GoogleImagenService, error) {
	ctx := context.Background()
	
	// Get credentials from environment
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsFile == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	// Create client
	client, err := aiplatform.NewPredictionClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	logrus.Info("Google Imagen service initialized")
	return &GoogleImagenService{
		projectID: projectID,
		location:  "us-central1",
		client:    client,
	}, nil
}

// GenerateImage generates an image using Google Imagen
func (g *GoogleImagenService) GenerateImage(prompt string) (string, error) {
	ctx := context.Background()

	// Create the request
	req := &aiplatformpb.PredictRequest{
		Endpoint: fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/imagen-3.0-generate-001", g.projectID, g.location),
		Instances: []*structpb.Value{
			{
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"prompt": {
								Kind: &structpb.Value_StringValue{
									StringValue: prompt,
								},
							},
							"number_of_images": {
								Kind: &structpb.Value_NumberValue{
									NumberValue: 1,
								},
							},
							"width": {
								Kind: &structpb.Value_NumberValue{
									NumberValue: 1024,
								},
							},
							"height": {
								Kind: &structpb.Value_NumberValue{
									NumberValue: 768,
								},
							},
						},
					},
				},
			},
		},
	}

	// Call Vertex AI
	resp, err := g.client.Predict(ctx, req)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate image with Imagen")
		return "", fmt.Errorf("failed to generate image: %w", err)
	}

	// Extract image from response
	if len(resp.Predictions) == 0 {
		return "", fmt.Errorf("no predictions returned")
	}

	// Parse the prediction - should be a struct with bytesBase64Encoded field
	prediction := resp.Predictions[0]
	if prediction.GetStructValue() == nil {
		logrus.WithField("prediction_type", fmt.Sprintf("%T", prediction.Kind)).Error("Invalid prediction format")
		return "", fmt.Errorf("invalid prediction format")
	}

	fields := prediction.GetStructValue().Fields
	
	// Log all available fields for debugging
	fieldNames := make([]string, 0, len(fields))
	for k := range fields {
		fieldNames = append(fieldNames, k)
	}
	logrus.WithField("response_fields", fieldNames).Info("Imagen API response fields")
	
	// Log the full prediction structure for debugging
	logrus.WithField("prediction_json", fmt.Sprintf("%+v", prediction)).Debug("Full prediction structure")

	// Get bytesBase64Encoded field (this is the standard Imagen API response format)
	imageBase64 := ""
	if bytesField := fields["bytesBase64Encoded"]; bytesField != nil {
		imageBase64 = bytesField.GetStringValue()
		logrus.WithField("bytes_length", len(imageBase64)).Info("Found bytesBase64Encoded field")
	}

	if imageBase64 == "" {
		// Log all field values for debugging
		for k, v := range fields {
			logrus.WithFields(logrus.Fields{
				"field_name": k,
				"field_type": fmt.Sprintf("%T", v.Kind),
				"field_value": fmt.Sprintf("%+v", v),
			}).Debug("Response field details")
		}
		logrus.WithField("available_fields", fieldNames).Error("No bytesBase64Encoded field in response")
		return "", fmt.Errorf("no image bytes in response")
	}

	logrus.Info("Image generated successfully with Google Imagen")

	return imageBase64, nil
}

// GenerateStoryImages generates multiple images for a story
func (g *GoogleImagenService) GenerateStoryImages(title, description string, geminiClient *GeminiClient) ([]string, error) {
	// Use Gemini to generate prompts
	prompts := g.generateImagePromptsWithGemini(title, description, geminiClient)
	
	var imageBase64s []string
	for _, prompt := range prompts {
		base64, err := g.GenerateImage(prompt)
		if err != nil {
			logrus.WithError(err).Warn("Failed to generate image")
			continue
		}
		imageBase64s = append(imageBase64s, base64)
	}

	return imageBase64s, nil
}

func (g *GoogleImagenService) generateImagePromptsWithGemini(title, description string, geminiClient *GeminiClient) []string {
	prompt := fmt.Sprintf(`Generate 2-3 detailed, specific image prompts for this news story.

HEADLINE: %s
DESCRIPTION: %s

Requirements:
- Each prompt should be 50-100 words
- Be specific to the story (not generic)
- Include composition, lighting, mood, style
- Suitable for professional news media

Format as JSON array:
["prompt 1", "prompt 2", "prompt 3"]

Return ONLY the JSON array.`, title, description)

	response, err := geminiClient.GenerateContent(prompt)
	if err != nil {
		logrus.WithError(err).Warn("Failed to generate prompts")
		return []string{fmt.Sprintf("Professional news illustration for: %s", title)}
	}

	var prompts []string
	err = json.Unmarshal([]byte(response), &prompts)
	if err != nil {
		return []string{response}
	}

	return prompts
}

func (g *GoogleImagenService) Close() error {
	return g.client.Close()
}
