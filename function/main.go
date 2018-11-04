package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/cloud-functions-go/nodego"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudiot/v1"
)

var (
	projectID   = os.Getenv("PROJECT_ID")
	cloudRegion = os.Getenv("CLOUD_REGION")
	registoryID = os.Getenv("REGISTORY_ID")
	deviceID    = os.Getenv("DEVICE_ID")
)

const (
	parentNameFormat   = "projects/%s/locations/%s"
	registryNameFormat = "/registries/%s"
	deviceNameFormat   = "/devices/%s"
)

func deviceName(projectID, region, registoryID, deviceID string) string {
	return fmt.Sprintf(parentNameFormat+registryNameFormat+deviceNameFormat, projectID, region, registoryID, deviceID)
}

func main() {
	flag.Parse()
	http.HandleFunc(nodego.HTTPTrigger, func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		client, err := google.DefaultClient(ctx, cloudiot.CloudPlatformScope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cloudiotService, err := cloudiot.New(client)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dn := deviceName(projectID, cloudRegion, registoryID, deviceID)
		data := base64.StdEncoding.EncodeToString([]byte("testData"))
		req := cloudiotService.Projects.Locations.Registries.Devices.SendCommandToDevice(dn, &cloudiot.SendCommandToDeviceRequest{
			BinaryData: data,
			Subfolder:  "test",
		})
		res, err := req.Do()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("status:%d\n", res.HTTPStatusCode)
	})

	nodego.TakeOver()
}
