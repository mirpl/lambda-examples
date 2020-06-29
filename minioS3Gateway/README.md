# Github Actions Envs:
- AWS_ACCESS_KEY_ID - AWS Access key
- AWS_SECRET_ACCESS_KEY - AWS Secret access key

# Lambda Envs:
- MINIO_ENDPOINT - MinIO gateway endpoint
- MINIO_ACCESS_KEY - access key for the object storage
- MINIO_SECRET_KEY - secret key for the object storage
- MINIO_USE_SSL - use secure https protocol or not (boolean)
- MINIO_REGION - region code where the bucket is to be created (if it doesn't exist); default: _us-east-1_
- MINIO_BUCKET - bucket name

## Test Deployment

#### Step 1: Run S3 gateway:

You need to have Docker installed on your machine. Once Docker installed, type the following commands in your terminal:

```
docker pull minio/minio
docker run -p 9000:9000 \
    -e "MINIO_ACCESS_KEY=<object storage access key>" \
    -e "MINIO_SECRET_KEY=<object storage secret key ID>" \
    minio/minio gateway s3
```

#### Step 2: Deploy with Serverless:

You need to have Serverless installed on your machine.
Once Serverless' installed, go to this README's location in your terminal and type the following commands:

- File Upload:

```
sls invoke local -f minioS3Gateway \
    -e MINIO_ENDPOINT=<gateway endpoint> \
    -e MINIO_ACCESS_KEY=<object storage access key> \
    -e MINIO_SECRET_KEY=<object storage secret key ID> \
    -e MINIO_USE_SSL=false \
    -e MINIO_REGION=us-east-1 \
    -e MINIO_BUCKET=lambda-example-file-storage \
    --path data_upload.json
```

- File Download:

```
sls invoke local -f minioS3Gateway \
    -e MINIO_ENDPOINT=<gateway endpoint> \
    -e MINIO_ACCESS_KEY=<object storage access key> \
    -e MINIO_SECRET_KEY=<object storage secret key ID> \
    -e MINIO_USE_SSL=false \
    -e MINIO_REGION=us-east-1 \
    -e MINIO_BUCKET=lambda-example-file-storage \
    --path data_download.json
```

