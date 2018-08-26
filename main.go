package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// Retrieve the public facing IP from the ip.42.pl site which ruturns plain
// text response
func GetMyIp(pipe chan string) {
	resp, err := http.Get("http://ip.42.pl/raw")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	pipe <- string(body)
}

// Query cloudflare for existing domain record
func getExistingRecord(api *cloudflare.API, pipe chan cloudflare.DNSRecord) {

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

	// Construct a new API object
	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	ipChannel := make(chan string)
	go GetMyIp(ipChannel)

	recordChannel := make(chan cloudflare.DNSRecord)
	go getExistingRecord(api, recordChannel)

	currentIP := <-ipChannel
	record := <-recordChannel

	// Update the IP if needed
	if currentIP != record.Content {
		log.Printf("IP has changed from %s to %s", record.Content, currentIP)
		record.Content = currentIP
		err = api.UpdateDNSRecord(record.ZoneID, record.ID, record)
		if err != nil {
			log.Fatal(err)
		}
	}

}
