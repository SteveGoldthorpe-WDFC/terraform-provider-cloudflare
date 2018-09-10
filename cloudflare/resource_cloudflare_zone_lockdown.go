package cloudflare

import (
	"fmt"
	"log"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceCloudflareZoneLockdown() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudflareZoneLockdownCreate,
		Read:   resourceCloudflareZoneLockdownRead,
		Update: resourceCloudflareZoneLockdownUpdate,
		Delete: resourceCloudflareZoneLockdownDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareZoneLockdownImport,
		},

		Schema: map[string]*schema.Schema{
			"zone": {
				Type:     schema.TypeString,
				Required: true,
			},
			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"paused": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 1024),
			},
			"urls": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"configurations": {
				Type:     schema.TypeSet,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"ip", "ip_range"}, false),
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceCloudflareZoneLockdownCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zone := d.Get("zone").(string)

	zoneID, err := client.ZoneIDByName(zone)
	if err != nil {
		return err
	}
	d.Set("zone_id", zoneID)

	var newZoneLockdown cloudflare.ZoneLockdown

	if paused, ok := d.GetOk("paused"); ok {
		newZoneLockdown.Paused = paused.(bool)
	}

	if description, ok := d.GetOk("description"); ok {
		newZoneLockdown.Description = description.(string)
	}

	if urls, ok := d.GetOk("urls"); ok {
		newZoneLockdown.URLs = expandInterfaceToStringList(urls.(*schema.Set).List())
	}

	if configurations, ok := d.GetOk("configurations"); ok {
		newZoneLockdown.Configurations = expandZoneLockdownConfig(configurations.(*schema.Set))
	}

	log.Printf("[INFO] Creating Cloudflare Zone Lockdown from struct: %+v", newZoneLockdown)

	var r *cloudflare.ZoneLockdownResponse

	r, err = client.CreateZoneLockdown(zoneID, newZoneLockdown)

	if err != nil {
		return fmt.Errorf("error creating zone lockdown for zone %q: %s", zone, err)
	}

	if r.Result.ID == "" {
		return fmt.Errorf("failed to find id in Create response; resource was empty")
	}

	d.SetId(r.Result.ID)

	log.Printf("[INFO] Cloudflare Zone Lockdown ID: %s", d.Id())

	return resourceCloudflareZoneLockdownRead(d, meta)
}

func resourceCloudflareZoneLockdownRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)

	log.Printf("[DEBUG] zoneID: %s", zoneID)
	zoneLockdownResponse, err := client.ZoneLockdown(zoneID, d.Id())

	log.Printf("[DEBUG] zoneLockdownResponse: %#v", zoneLockdownResponse)
	log.Printf("[DEBUG] zoneLockdownResponse error: %#v", err)

	if err != nil {
		if strings.Contains(err.Error(), "HTTP status 404") {
			log.Printf("[INFO] Zone Lockdown %s no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error finding zone lockdown %q: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Cloudflare Zone Lockdown read configuration: %#v", zoneLockdownResponse)

	d.Set("paused", zoneLockdownResponse.Result.Paused)
	d.Set("description", zoneLockdownResponse.Result.Description)
	d.Set("urls", zoneLockdownResponse.Result.URLs)
	log.Printf("[DEBUG] read configurations: %#v", d.Get("configurations"))

	configurations := make([]map[string]interface{}, len(zoneLockdownResponse.Result.Configurations))

	for i, entryconfigZoneLockdownConfig := range zoneLockdownResponse.Result.Configurations {
		configurations[i] = map[string]interface{}{
			"target": entryconfigZoneLockdownConfig.Target,
			"value":  entryconfigZoneLockdownConfig.Value,
		}
	}
	log.Printf("[DEBUG] Cloudflare Zone Lockdown configuration: %#v", configurations)

	if err := d.Set("configurations", configurations); err != nil {
		log.Printf("[WARN] Error setting configurations in zone lockdown %q: %s", d.Id(), err)
	}

	return nil
}

func resourceCloudflareZoneLockdownUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)

	newRule := cloudflare.AccessRule{
		Notes: d.Get("notes").(string),
		Mode:  d.Get("mode").(string),
	}

	if configuration, configurationOk := d.GetOk("configuration"); configurationOk {
		config := configuration.(map[string]interface{})

		newRule.Configuration = cloudflare.AccessRuleConfiguration{
			Target: config["target"].(string),
			Value:  config["value"].(string),
		}
	}

	// var accessRuleResponse *cloudflare.AccessRuleResponse
	var err error

	if zoneID == "" {
		if client.OrganizationID != "" {
			_, err = client.UpdateOrganizationAccessRule(client.OrganizationID, d.Id(), newRule)
		} else {
			_, err = client.UpdateUserAccessRule(d.Id(), newRule)
		}
	} else {
		_, err = client.UpdateZoneAccessRule(zoneID, d.Id(), newRule)
	}

	if err != nil {
		return fmt.Errorf("Failed to update Access Rule: %s", err)
	}

	return resourceCloudflareZoneLockdownRead(d, meta)
}

func resourceCloudflareZoneLockdownDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)

	log.Printf("[INFO] Deleting Cloudflare Zone Lockdown: id %s for zone_id %s", d.Id(), zoneID)

	var err error

	_, err = client.DeleteZoneLockdown(zoneID, d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting Cloudflare Zone Lockdown: %s", err)
	}

	return nil
}

func expandZoneLockdownConfig(configs *schema.Set) []cloudflare.ZoneLockdownConfig {
	configArray := make([]cloudflare.ZoneLockdownConfig, configs.Len())
	for i, entry := range configs.List() {
		e := entry.(map[string]interface{})
		configArray[i] = cloudflare.ZoneLockdownConfig{
			Target: e["target"].(string),
			Value:  e["value"].(string),
		}
	}
	return configArray
}

func resourceCloudflareZoneLockdownImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*cloudflare.API)

	// split the id so we can lookup
	idAttr := strings.SplitN(d.Id(), "/", 2)
	var zoneName string
	var zoneLockdownId string
	if len(idAttr) == 2 {
		zoneName = idAttr[0]
		zoneLockdownId = idAttr[1]
		d.Set("zone", zoneName)
		d.SetId(zoneLockdownId)
	} else {
		return nil, fmt.Errorf("invalid id (%q) specified, should be in format \"zoneName/zoneLockdownId\"", d.Id())
	}
	zoneId, err := client.ZoneIDByName(zoneName)
	d.Set("zone_id", zoneId)
	if err != nil {
		return nil, fmt.Errorf("couldn't find zone %q while trying to import zone lockdown %q : %q", zoneName, d.Id(), err)
	}
	log.Printf("[DEBUG] zone: %s", zoneName)
	log.Printf("[DEBUG] zoneID: %s", zoneId)
	log.Printf("[DEBUG] Resource ID : %s", zoneLockdownId)
	return []*schema.ResourceData{d}, nil
}
