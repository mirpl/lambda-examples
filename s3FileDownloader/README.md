# Github Actions Envs:
- AWS_ACCESS_KEY_ID - AWS Access key
- AWS_SECRET_ACCESS_KEY - AWS Secret access key

## Lambda Envs:
- S3_ENDPOINT - S3 compatible object storage endpoint, e.g. _s3.amazonaws.com_
- S3_ACCESS_KEY - access key for the object storage
- S3_SECRET_KEY - secret key for the object storage
- S3_REGION - region code, e.g. _us-east-1_
- S3_BUCKET - bucket name


## Test Deployment

#### Deploy with Serverless:

You need to have Serverless installed on your machine.
Once Serverless' installed, go to this README's location in your terminal and type the following command:

```
sls invoke local -f s3FileDownloader \
    -e S3_ENDPOINT=<object storage endpoint> \
    -e S3_ACCESS_KEY=<object storage access key> \
    -e S3_SECRET_KEY=<object storage secret key ID> \
    -e S3_REGION=us-east-1 \
    -e S3_BUCKET=lambda-example-file-storage \
    --path data.json
```
