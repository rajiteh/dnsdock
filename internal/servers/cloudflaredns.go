package servers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// CloudFlareDNSManager is a reusable struct that manages DNS operations for a specific Cloudflare zone.
type CloudFlareDNSManager struct {
	api      *cloudflare.API
	zoneID   *cloudflare.ResourceContainer
	zoneName string
}

// NewCloudflareDNSManager initializes a DNSManager for the specified zone using the provided API token.
// It fetches the zone ID once and reuses it for subsequent operations.
func NewCloudflareDNSManager(apiToken, zoneName string) (*CloudFlareDNSManager, error) {
	// Initialize the Cloudflare API client.
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
	}

	// Get the zone ID for the specified zone name.
	zoneID, err := api.ZoneIDByName(zoneName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID for %s: %w", zoneName, err)
	}

	return &CloudFlareDNSManager{
		api:      api,
		zoneID:   cloudflare.ZoneIdentifier(zoneID),
		zoneName: zoneName,
	}, nil
}

// UpdateARecord updates or creates an "A" DNS record in the managed zone.
// Parameters:
// - recordName: The name of the DNS record to update (e.g., "sub.example.com").
// - ipAddress: The IP address to set for the "A" record.
// - ttl: The Time-To-Live (TTL) for the DNS record.
func (d *CloudFlareDNSManager) UpdateARecord(recordName, ipAddress string, ttl int) error {

	if strings.HasSuffix(recordName, "ls90") {
		recordName = recordName + ".co"
	}

	if !strings.HasSuffix(recordName, d.zoneName) {
		logger.Warningf("Ignoring record %s: does not being to the zone", recordName)
		return nil
	}

	if ttl < 1 {
		ttl = 60 // Default TTL of 60 seconds.
	}

	// Check if the DNS record already exists.
	ctx := context.Background()
	records, _, err := d.api.ListDNSRecords(ctx, d.zoneID, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: recordName,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch DNS records: %w", err)
	}

	if len(records) > 0 {
		// Update the existing record if found.
		if records[0].Content == ipAddress {
			logger.Debugf("A record %s already up-to-date", recordName)
			return nil
		}
		_, err = d.api.UpdateDNSRecord(ctx, d.zoneID, cloudflare.UpdateDNSRecordParams{
			ID:      records[0].ID,
			Type:    "A",
			Name:    recordName,
			Content: ipAddress,
			TTL:     ttl,
		})
		if err != nil {
			return fmt.Errorf("failed to update DNS record: %w", err)
		}
		logger.Infof("Updated A record: %s -> %s", recordName, ipAddress)
	} else {
		// Create a new record if no existing record is found.
		newRecord := cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    recordName,
			Content: ipAddress,
			TTL:     ttl,
		}

		_, err = d.api.CreateDNSRecord(ctx, d.zoneID, newRecord)
		if err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}
		logger.Infof("Created new A record: %s -> %s", recordName, ipAddress)
	}

	return nil
}
