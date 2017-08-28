package main

import (
	"github.com/cloudflare/cloudflare-go"
	"log"
	"os"
	"net/http"
	"io/ioutil"
)

// Retrieve the public facing IP from the ip.42.pl site which ruturns plain text response
func GetMyIp() (string) {
	resp, err := http.Get("http://ip.42.pl/raw")

	if err != nil {
    log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
  return string(body)
}


func main() {

	currentIP := GetMyIp()

	// Construct a new API object
	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatal(err)
	}

	zoneID, err := api.ZoneIDByName(os.Getenv("CF_DNS_ZONE"))
	zoneID, err := api.ZoneIDByName("mattwells.ninja")
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

	// Update the IP if needed
  if currentIP != record.Content {
    log.Printf("IP has changed from %s to %s", record.Content, currentIP)
	  record.Content = currentIP
	  err = api.UpdateDNSRecord(zoneID,record.ID,record)
		if err != nil {
      log.Fatal(err)
		}
	}

}
