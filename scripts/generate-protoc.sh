#!/bin/bash -e

protoc -I driver/ driver/drivers.proto --go_out=plugins=grpc:driver