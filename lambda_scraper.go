package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"golang.org/x/net/html"
)

func scrape(url string, ch chan FoundWord, chFinished chan bool, words []string) {
	resp, err := http.Get(url)

	defer func() {
		// Finish sraping
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("Error scraping: " + url)
		fmt.Println(err)
		// Notify error by email
		notifyError(err, url)
		return
	}

	body := resp.Body
	defer body.Close()

	z := html.NewTokenizer(body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of html
			return
		case tt == html.TextToken:
			t := z.Token()

			// Check if words exist
			for _, word := range words {
				hasWord := strings.Contains(t.Data, word)
				if hasWord {
					ch <- FoundWord{word, url}
				}
			}
		}
	}
}

func notifyError(err error, url string) {
	emailSubject := fmt.Sprintf("Error parsing %s", url)
	emailTextBody := fmt.Sprintf("There was an error parsing: %s\n\n", err)

	// Send email
	sendEmail(emailSubject, emailTextBody)
}

func notifyResults(words []string, url string) {
	// Build email args
	emailSubject := fmt.Sprintf("Found %d words on %s", len(words), url)
	emailTextBody := fmt.Sprintf("Words found on %s\n", url)

	for _, word := range words {
		emailTextBody = emailTextBody + ("\n - " + word)
	}

	// Send email
	sendEmail(emailSubject, emailTextBody)
}

func sendEmail(emailSubject string, emailTextBody string) {
	// Get sender/recipient from ENV
	sender := os.Getenv("SENDER")
	recipient := os.Getenv("RECIPIENT")

	sesLocation := os.Getenv("SES_LOCATION")

	// The character encoding for the email.
	emailCharSet := "UTF-8"

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(sesLocation)},
	)

	// Create SES session
	svc := ses.New(sess)

	// Build email
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(emailCharSet),
					Data:    aws.String(emailTextBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(emailCharSet),
				Data:    aws.String(emailSubject),
			},
		},
		Source: aws.String(sender),
	}

	// Attempt to send the email
	result, err := svc.SendEmail(input)

	// Print any error messages
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println("Email Sent to address: " + recipient)
	fmt.Println(result)
}

// FoundWord represents words found during scraping
type FoundWord struct {
	word string
	url  string
}

func start(event scrapeData) {
	urlsToScrape := event.Urls
	containsWords := event.Words

	urlsToScrapeArray := strings.Split(urlsToScrape, ",")
	containsWordsArray := strings.Split(containsWords, ",")
	results := make(map[string][]string)

	// Create channels
	chUrls := make(chan FoundWord)
	chFinished := make(chan bool)

	// Start scrape for urls
	for _, url := range urlsToScrapeArray {
		go scrape(url, chUrls, chFinished, containsWordsArray)
	}

	// Subscribe to scrape channels
	for c := 0; c < len(urlsToScrapeArray); {
		select {
		case word := <-chUrls:
			if contains(results[word.url], word.word) == false {
				results[word.url] = append(results[word.url], word.word)
			}
		case <-chFinished:
			c++
		}
	}

	// Finish, print results
	for url, words := range results {
		fmt.Println("\nFound", len(words), "words on", url)
		for _, word := range words {
			fmt.Println(" - " + word)
		}

		if len(words) > 0 {
			// We found words, let's notify with SES
			notifyResults(words, url)
		}
	}

	close(chUrls)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type scrapeData struct {
	Urls  string "json:urls"
	Words string "json:words"
}

func main() {
	lambda.Start(start)
}
