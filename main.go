package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

var (
	Info    *log.Logger
	Warning *log.Logger
)

func SendUpdateSMS(message string) {
	sess := session.Must(session.NewSession())
	svc := sns.New(sess)

	atrs := map[string]*sns.MessageAttributeValue{}
	atrs["AWS.SNS.SMS.MaxPrice"] = &sns.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String("1.0"),
	}
	atrs["AWS.SNS.SMS.SenderID"] = &sns.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String("Updatr"),
	}

	params := &sns.PublishInput{
		Message:           aws.String(message),
		PhoneNumber:       aws.String("+447739250216"),
		MessageAttributes: atrs,
	}
	resp, err := svc.Publish(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		Info.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	Info.Println(resp)
}

// Retrieve the public facing IP from the ip.42.pl site which ruturns plain
// text response
func GetMyIp(pipe chan string) {
	message := make(chan string, 1)

	go func() {
		resp, err := http.Get("http://ip.42.pl/raw")

		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		message <- string(body)
	}()

	select {
	case resp := <-message:
		pipe <- resp
	case <-time.After(2 * time.Second):
		log.Fatal("Call to fetch IP timed out")
	}

}

// Query cloudflare for existing domain record
func GetExistingRecord(api *cloudflare.API, pipe chan cloudflare.DNSRecord) {

	zoneID, err := api.ZoneIDByName(os.Getenv("CF_DNS_ZONE"))
	if err != nil {
		log.Fatal(err)
	}

	// Fetch all records for a zone
	recs, err := api.DNSRecords(zoneID, cloudflare.DNSRecord{})
	if err != nil {
		log.Fatal(err)
	}

	// determine the A record
	var record cloudflare.DNSRecord
	for _, r := range recs {
		if r.Type == "A" {
			record = r
		}
	}

	pipe <- record
}

func main() {

	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	// Construct a new API object
	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	ipChannel := make(chan string, 1)
	recordChannel := make(chan cloudflare.DNSRecord, 1)

	go GetMyIp(ipChannel)
	go GetExistingRecord(api, recordChannel)

	currentIP := <-ipChannel
	record := <-recordChannel

	// Update the IP if needed
	if currentIP != record.Content {
		message := fmt.Sprintf("IP has changed from %s to %s", record.Content, currentIP)
		Info.Println(message)
		record.Content = currentIP
		err = api.UpdateDNSRecord(record.ZoneID, record.ID, record)
		if err != nil {
			log.Fatal(err)
		}

		SendUpdateSMS(message)
	}

}
