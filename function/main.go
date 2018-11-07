package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/cloud-functions-go/nodego"
	"github.com/ww24/cloud-iot-mqtt/payload"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudiot/v1"
)

var (
	projectID   = os.Getenv("PROJECT_ID")
	cloudRegion = os.Getenv("CLOUD_REGION")
	registoryID = os.Getenv("REGISTORY_ID")
)

const (
	parentNameFormat   = "projects/%s/locations/%s"
	registryNameFormat = "/registries/%s"
	deviceNameFormat   = "/devices/%s"
)

type request struct {
	DeviceID string `json:"device_id"`
	payload.Payload
}

type response struct {
	Message string `json:"message"`
}

func responseJSON(msg string) string {
	res := &response{Message: msg}
	d, _ := json.Marshal(res)
	return string(d)
}

func deviceName(projectID, region, registoryID, deviceID string) string {
	return fmt.Sprintf(parentNameFormat+registryNameFormat+deviceNameFormat, projectID, region, registoryID, deviceID)
}

func init() {
	nodego.OverrideLogger()
}

func main() {
	flag.Parse()

	http.HandleFunc(nodego.HTTPTrigger, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				nodego.ErrorLogger.Printf("%+v", r)
			}
		}()

		// set cors headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Vary, Origin, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "300")

		switch r.Method {
		case http.MethodOptions:
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodPost:
			// allow only post
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		req := &request{}
		if r.Body != nil {
			decoder := json.NewDecoder(r.Body)
			defer r.Body.Close()
			if err := decoder.Decode(req); err != nil {
				nodego.ErrorLogger.Printf("%+v", err)
				http.Error(w, responseJSON(err.Error()), http.StatusInternalServerError)
				return
			}
		}
		reqJSON, _ := json.Marshal(req)
		log.Println("Request:", string(reqJSON))

		ctx := context.Background()
		client, err := google.DefaultClient(ctx, cloudiot.CloudPlatformScope)
		if err != nil {
			nodego.ErrorLogger.Printf("%+v", err)
			http.Error(w, responseJSON(err.Error()), http.StatusInternalServerError)
			return
		}

		cloudiotService, err := cloudiot.New(client)
		if err != nil {
			nodego.ErrorLogger.Printf("%+v", err)
			http.Error(w, responseJSON(err.Error()), http.StatusInternalServerError)
			return
		}

		payload, err := json.Marshal(&req.Payload)
		if err != nil {
			nodego.ErrorLogger.Printf("%+v", err)
			http.Error(w, responseJSON(err.Error()), http.StatusInternalServerError)
			return
		}

		dn := deviceName(projectID, cloudRegion, registoryID, req.DeviceID)
		iotReq := cloudiotService.Projects.Locations.Registries.Devices.SendCommandToDevice(dn, &cloudiot.SendCommandToDeviceRequest{
			BinaryData: base64.StdEncoding.EncodeToString(payload),
			Subfolder:  "signal",
		})
		res, err := iotReq.Do()
		if err != nil {
			nodego.ErrorLogger.Printf("%+v", err)
			http.Error(w, responseJSON(err.Error()), http.StatusInternalServerError)
			return
		}

		log.Printf("Status:%d", res.HTTPStatusCode)
		fmt.Fprintln(w, responseJSON("success"))
	})

	nodego.TakeOver()
}
