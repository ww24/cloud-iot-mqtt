
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
