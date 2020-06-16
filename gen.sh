#!/bin/bash

for i in {100..199}; do
	gcloud iam service-accounts create rclone$i --display-name "rclone$i"
	gcloud iam service-accounts keys create ~/.config/rclone/rclone-api-service${i}.json --iam-account rclone$i@rclone-api-service2.iam.gserviceaccount.com
done
