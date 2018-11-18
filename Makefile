
include env.mk
ENV := BROKER=$(BROKER) PROJECT_ID=$(PROJECT_ID) CLOUD_REGION=$(CLOUD_REGION) REGISTORY_ID=$(REGISTORY_ID) DEVICE_ID=$(DEVICE_ID) ENDPOINT=$(ENDPOINT)

.PHONY: run
run:
	@$(ENV) go run main.go

.PHONY: dep
dep:
	dep ensure -v

.PHONY: function
function:
	@cd function; make

.PHONY: build
build: dep
	CGO_ENABLED=0 GOARCH=arm GOARM=5 GOOS=linux go build -o iot_client .
	@echo "BROKER=$(BROKER) PROJECT_ID=$(PROJECT_ID) CLOUD_REGION=$(CLOUD_REGION) REGISTORY_ID=$(REGISTORY_ID) DEVICE_ID=$(DEVICE_ID) ENDPOINT=$(ENDPOINT) ./iot_client > out.log 2>&1 &" > run.sh
	@chmod +x run.sh
	zip -FS -r deploy_with_secret.zip iot_client run.sh *.pem
	@rm run.sh

.PHONY: run-bin
run-bin:
	@$(ENV) ./iot_client
