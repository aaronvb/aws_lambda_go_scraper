# Scrape websites with AWS Lambda
This is a basic website scraper written in Go that will search for text.

If text is found, results are e-mailed to user through Amazon SES.

## Why
I built this to let me know when certain websites update with the text I'm looking for. This utilizes AWS Lambda and cron to run continuously without a server setup.

## How to use this

### Setup
First, clone the repo.
```
> git clone git@github.com:aaronvb/aws_lambda_go_scraper.git
> cd aws_lambda_go_scraper
```

Build the Go script and zip for AWS Lambda.
```
> GOOS=linux GOARCH=amd64 go build -o main lambda_scraper.go
> zip main.zip main
```

Upload the zip file to the AWS Lambda function, and make sure the handler is set to `main`.

Next, create 3 environment variables: `RECIPIENT` will be the email which receives the notification, `SENDER` which will be the email address that sends the notification, and last `SES_LOCATION` which is the location of your SES(ie: us-west-2).

Finally, make sure the role which the AWS Lambda function is using has permission to Amazon SES. Also, don't forget to add the email address to SES and verify it so it can receive emails.

### Running the function
Create a test event. In the event data pass a JSON hash which has a key `urls` and a string value with the urls you want to scrape, separated by commas, and a key `words`, with a string value of comma separated words you wish to scrape.

Example:

```
{
  "urls": "https://aaronvb.com,https://aaronvb.com/articles/selection-sort-in-ruby.html",
  "words": "ruby,Hawaii,foobar"
}
```

## Automation

I [wrote an article](https://medium.com/@aaronvb/simple-website-text-scraping-with-go-and-aws-lambda-cd5df25f5b2b) explaining how to setup automated scraping using AWS CloudWatch with AWS Lambda.