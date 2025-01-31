package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	tfcloudtrail "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/cloudtrail"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
)

func resourceAwsCloudTrail() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudTrailCreate,
		Read:   resourceAwsCloudTrailRead,
		Update: resourceAwsCloudTrailUpdate,
		Delete: resourceAwsCloudTrailDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"advanced_event_selector": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"event_selector"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field_selector": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ends_with": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
									"equals": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
									"field": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(tfcloudtrail.Field_Values(), false),
									},
									"not_ends_with": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
									"not_equals": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
									"not_starts_with": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
									"starts_with": {
										Type:     schema.TypeList,
										Optional: true,
										MinItems: 1,
										Elem: &schema.Schema{
											Type:         schema.TypeString,
											ValidateFunc: validation.StringLenBetween(1, 2048),
										},
									},
								},
							},
						},
						"name": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(0, 1000),
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cloud_watch_logs_group_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cloud_watch_logs_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enable_log_file_validation": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"enable_logging": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"event_selector": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      5,
				ConflictsWith: []string{"advanced_event_selector"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"data_resource": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(tfcloudtrail.ResourceType_Values(), false),
									},
									"values": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 250,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"include_management_events": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"read_write_type": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      cloudtrail.ReadWriteTypeAll,
							ValidateFunc: validation.StringInSlice(cloudtrail.ReadWriteType_Values(), false),
						},
					},
				},
			},
			"home_region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"include_global_service_events": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"insight_selector": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"insight_type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(cloudtrail.InsightType_Values(), false),
						},
					},
				},
			},
			"is_multi_region_trail": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"is_organization_trail": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"s3_bucket_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"s3_key_prefix": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"sns_topic_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsCloudTrailCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := cloudtrail.CreateTrailInput{
		Name:         aws.String(d.Get("name").(string)),
		S3BucketName: aws.String(d.Get("s3_bucket_name").(string)),
	}

	if len(tags) > 0 {
		input.TagsList = tags.IgnoreAws().CloudtrailTags()
	}

	if v, ok := d.GetOk("cloud_watch_logs_group_arn"); ok {
		input.CloudWatchLogsLogGroupArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("cloud_watch_logs_role_arn"); ok {
		input.CloudWatchLogsRoleArn = aws.String(v.(string))
	}
	if v, ok := d.GetOkExists("include_global_service_events"); ok {
		input.IncludeGlobalServiceEvents = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("is_multi_region_trail"); ok {
		input.IsMultiRegionTrail = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("is_organization_trail"); ok {
		input.IsOrganizationTrail = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("enable_log_file_validation"); ok {
		input.EnableLogFileValidation = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("kms_key_id"); ok {
		input.KmsKeyId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("s3_key_prefix"); ok {
		input.S3KeyPrefix = aws.String(v.(string))
	}
	if v, ok := d.GetOk("sns_topic_name"); ok {
		input.SnsTopicName = aws.String(v.(string))
	}

	var t *cloudtrail.CreateTrailOutput
	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error
		t, err = conn.CreateTrail(&input)
		if err != nil {
			if isAWSErr(err, cloudtrail.ErrCodeInvalidCloudWatchLogsRoleArnException, "Access denied.") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, cloudtrail.ErrCodeInvalidCloudWatchLogsLogGroupArnException, "Access denied.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		t, err = conn.CreateTrail(&input)
	}
	if err != nil {
		return fmt.Errorf("Error creating CloudTrail: %s", err)
	}

	log.Printf("[DEBUG] CloudTrail created: %s", t)

	d.SetId(aws.StringValue(t.Name))

	// AWS CloudTrail sets newly-created trails to false.
	if v, ok := d.GetOk("enable_logging"); ok && v.(bool) {
		err := cloudTrailSetLogging(conn, v.(bool), d.Id())
		if err != nil {
			return err
		}
	}

	// Event Selectors
	if _, ok := d.GetOk("event_selector"); ok {
		if err := cloudTrailSetEventSelectors(conn, d); err != nil {
			return err
		}
	}

	if _, ok := d.GetOk("advanced_event_selector"); ok {
		if err := cloudTrailSetAdvancedEventSelectors(conn, d); err != nil {
			return err
		}
	}

	if _, ok := d.GetOk("insight_selector"); ok {
		if err := cloudTrailSetInsightSelectors(conn, d); err != nil {
			return err
		}
	}

	return resourceAwsCloudTrailRead(d, meta)
}

func resourceAwsCloudTrailRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	input := cloudtrail.DescribeTrailsInput{
		TrailNameList: []*string{
			aws.String(d.Id()),
		},
	}
	resp, err := conn.DescribeTrails(&input)
	if err != nil {
		return err
	}

	// CloudTrail does not return a NotFound error in the event that the Trail
	// you're looking for is not found. Instead, it's simply not in the list.
	var trail *cloudtrail.Trail
	for _, c := range resp.TrailList {
		if d.Id() == aws.StringValue(c.Name) {
			trail = c
		}
	}

	if trail == nil {
		log.Printf("[WARN] CloudTrail (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] CloudTrail received: %s", trail)

	d.Set("name", trail.Name)
	d.Set("s3_bucket_name", trail.S3BucketName)
	d.Set("s3_key_prefix", trail.S3KeyPrefix)
	d.Set("cloud_watch_logs_role_arn", trail.CloudWatchLogsRoleArn)
	d.Set("cloud_watch_logs_group_arn", trail.CloudWatchLogsLogGroupArn)
	d.Set("include_global_service_events", trail.IncludeGlobalServiceEvents)
	d.Set("is_multi_region_trail", trail.IsMultiRegionTrail)
	d.Set("is_organization_trail", trail.IsOrganizationTrail)
	d.Set("sns_topic_name", trail.SnsTopicName)
	d.Set("enable_log_file_validation", trail.LogFileValidationEnabled)

	// TODO: Make it possible to use KMS Key names, not just ARNs
	// In order to test it properly this PR needs to be merged 1st:
	// https://github.com/hashicorp/terraform/pull/3928
	d.Set("kms_key_id", trail.KmsKeyId)

	d.Set("arn", trail.TrailARN)
	d.Set("home_region", trail.HomeRegion)

	tags, err := keyvaluetags.CloudtrailListTags(conn, *trail.TrailARN)

	if err != nil {
		return fmt.Errorf("error listing tags for Cloudtrail (%s): %s", *trail.TrailARN, err)
	}

	tags = tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	logstatus, err := cloudTrailGetLoggingStatus(conn, trail.Name)
	if err != nil {
		return err
	}
	d.Set("enable_logging", logstatus)

	// Get EventSelectors
	eventSelectorsOut, err := conn.GetEventSelectors(&cloudtrail.GetEventSelectorsInput{
		TrailName: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	if err := d.Set("event_selector", flattenAwsCloudTrailEventSelector(eventSelectorsOut.EventSelectors)); err != nil {
		return err
	}

	if err := d.Set("advanced_event_selector", flattenAwsCloudTrailAdvancedEventSelector(eventSelectorsOut.AdvancedEventSelectors)); err != nil {
		return err
	}

	// Get InsightSelectors
	insightSelectors, err := conn.GetInsightSelectors(&cloudtrail.GetInsightSelectorsInput{
		TrailName: aws.String(d.Id()),
	})
	if err != nil {
		if !isAWSErr(err, cloudtrail.ErrCodeInsightNotEnabledException, "") {
			return fmt.Errorf("error getting Cloud Trail (%s) Insight Selectors: %w", d.Id(), err)
		}
	}
	if insightSelectors != nil {
		if err := d.Set("insight_selector", flattenAwsCloudTrailInsightSelector(insightSelectors.InsightSelectors)); err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsCloudTrailUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn

	input := cloudtrail.UpdateTrailInput{
		Name: aws.String(d.Id()),
	}

	if d.HasChange("s3_bucket_name") {
		input.S3BucketName = aws.String(d.Get("s3_bucket_name").(string))
	}
	if d.HasChange("s3_key_prefix") {
		input.S3KeyPrefix = aws.String(d.Get("s3_key_prefix").(string))
	}
	if d.HasChanges("cloud_watch_logs_role_arn", "cloud_watch_logs_group_arn") {
		// Both of these need to be provided together
		// in the update call otherwise API complains
		input.CloudWatchLogsRoleArn = aws.String(d.Get("cloud_watch_logs_role_arn").(string))
		input.CloudWatchLogsLogGroupArn = aws.String(d.Get("cloud_watch_logs_group_arn").(string))
	}
	if d.HasChange("include_global_service_events") {
		input.IncludeGlobalServiceEvents = aws.Bool(d.Get("include_global_service_events").(bool))
	}
	if d.HasChange("is_multi_region_trail") {
		input.IsMultiRegionTrail = aws.Bool(d.Get("is_multi_region_trail").(bool))
	}
	if d.HasChange("is_organization_trail") {
		input.IsOrganizationTrail = aws.Bool(d.Get("is_organization_trail").(bool))
	}
	if d.HasChange("enable_log_file_validation") {
		input.EnableLogFileValidation = aws.Bool(d.Get("enable_log_file_validation").(bool))
	}
	if d.HasChange("kms_key_id") {
		input.KmsKeyId = aws.String(d.Get("kms_key_id").(string))
	}
	if d.HasChange("sns_topic_name") {
		input.SnsTopicName = aws.String(d.Get("sns_topic_name").(string))
	}

	log.Printf("[DEBUG] Updating CloudTrail: %s", input)
	var t *cloudtrail.UpdateTrailOutput
	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error
		t, err = conn.UpdateTrail(&input)
		if err != nil {
			if isAWSErr(err, cloudtrail.ErrCodeInvalidCloudWatchLogsRoleArnException, "Access denied.") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, cloudtrail.ErrCodeInvalidCloudWatchLogsLogGroupArnException, "Access denied.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		t, err = conn.UpdateTrail(&input)
	}
	if err != nil {
		return fmt.Errorf("Error updating CloudTrail: %s", err)
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.CloudtrailUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating ECR Repository (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	if d.HasChange("enable_logging") {
		log.Printf("[DEBUG] Updating logging on CloudTrail: %s", input)
		err := cloudTrailSetLogging(conn, d.Get("enable_logging").(bool), *input.Name)
		if err != nil {
			return err
		}
	}

	if !d.IsNewResource() && d.HasChange("event_selector") {
		log.Printf("[DEBUG] Updating event selector on CloudTrail: %s", input)
		if err := cloudTrailSetEventSelectors(conn, d); err != nil {
			return err
		}
	}

	if !d.IsNewResource() && d.HasChange("advanced_event_selector") {
		log.Printf("[DEBUG] Updating advanced event selector on CloudTrail: %s", input)
		if err := cloudTrailSetAdvancedEventSelectors(conn, d); err != nil {
			return err
		}
	}

	if !d.IsNewResource() && d.HasChange("insight_selector") {
		log.Printf("[DEBUG] Updating insight selector on CloudTrail: %s", input)
		if err := cloudTrailSetInsightSelectors(conn, d); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] CloudTrail updated: %s", t)

	return resourceAwsCloudTrailRead(d, meta)
}

func resourceAwsCloudTrailDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn

	log.Printf("[DEBUG] Deleting CloudTrail: %q", d.Id())
	_, err := conn.DeleteTrail(&cloudtrail.DeleteTrailInput{
		Name: aws.String(d.Id()),
	})

	return err
}

func cloudTrailGetLoggingStatus(conn *cloudtrail.CloudTrail, id *string) (bool, error) {
	GetTrailStatusOpts := &cloudtrail.GetTrailStatusInput{
		Name: id,
	}
	resp, err := conn.GetTrailStatus(GetTrailStatusOpts)
	if err != nil {
		return false, fmt.Errorf("Error retrieving logging status of CloudTrail (%s): %s", *id, err)
	}

	return *resp.IsLogging, err
}

func cloudTrailSetLogging(conn *cloudtrail.CloudTrail, enabled bool, id string) error {
	if enabled {
		log.Printf(
			"[DEBUG] Starting logging on CloudTrail (%s)",
			id)
		StartLoggingOpts := &cloudtrail.StartLoggingInput{
			Name: aws.String(id),
		}
		if _, err := conn.StartLogging(StartLoggingOpts); err != nil {
			return fmt.Errorf(
				"Error starting logging on CloudTrail (%s): %s",
				id, err)
		}
	} else {
		log.Printf(
			"[DEBUG] Stopping logging on CloudTrail (%s)",
			id)
		StopLoggingOpts := &cloudtrail.StopLoggingInput{
			Name: aws.String(id),
		}
		if _, err := conn.StopLogging(StopLoggingOpts); err != nil {
			return fmt.Errorf(
				"Error stopping logging on CloudTrail (%s): %s",
				id, err)
		}
	}

	return nil
}

func cloudTrailSetEventSelectors(conn *cloudtrail.CloudTrail, d *schema.ResourceData) error {
	input := &cloudtrail.PutEventSelectorsInput{
		TrailName: aws.String(d.Id()),
	}

	eventSelectors := expandAwsCloudTrailEventSelector(d.Get("event_selector").([]interface{}))
	// If no defined selectors revert to the single default selector
	if len(eventSelectors) == 0 {
		es := &cloudtrail.EventSelector{
			IncludeManagementEvents: aws.Bool(true),
			ReadWriteType:           aws.String("All"),
			DataResources:           make([]*cloudtrail.DataResource, 0),
		}
		eventSelectors = append(eventSelectors, es)
	}
	input.EventSelectors = eventSelectors

	if err := input.Validate(); err != nil {
		return fmt.Errorf("Error validate CloudTrail (%s): %s", d.Id(), err)
	}

	_, err := conn.PutEventSelectors(input)
	if err != nil {
		return fmt.Errorf("Error set event selector on CloudTrail (%s): %s", d.Id(), err)
	}

	return nil
}

func expandAwsCloudTrailEventSelector(configured []interface{}) []*cloudtrail.EventSelector {
	eventSelectors := make([]*cloudtrail.EventSelector, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})
		dataResources := expandAwsCloudTrailEventSelectorDataResource(data["data_resource"].([]interface{}))

		es := &cloudtrail.EventSelector{
			IncludeManagementEvents: aws.Bool(data["include_management_events"].(bool)),
			ReadWriteType:           aws.String(data["read_write_type"].(string)),
			DataResources:           dataResources,
		}
		eventSelectors = append(eventSelectors, es)
	}

	return eventSelectors
}

func expandAwsCloudTrailEventSelectorDataResource(configured []interface{}) []*cloudtrail.DataResource {
	dataResources := make([]*cloudtrail.DataResource, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		values := make([]*string, len(data["values"].([]interface{})))
		for i, vv := range data["values"].([]interface{}) {
			str := vv.(string)
			values[i] = aws.String(str)
		}

		dataResource := &cloudtrail.DataResource{
			Type:   aws.String(data["type"].(string)),
			Values: values,
		}

		dataResources = append(dataResources, dataResource)
	}

	return dataResources
}

func flattenAwsCloudTrailEventSelector(configured []*cloudtrail.EventSelector) []map[string]interface{} {
	eventSelectors := make([]map[string]interface{}, 0, len(configured))

	// Prevent default configurations shows differences
	if len(configured) == 1 && len(configured[0].DataResources) == 0 && aws.StringValue(configured[0].ReadWriteType) == "All" {
		return eventSelectors
	}

	for _, raw := range configured {
		item := make(map[string]interface{})
		item["read_write_type"] = aws.StringValue(raw.ReadWriteType)
		item["include_management_events"] = aws.BoolValue(raw.IncludeManagementEvents)
		item["data_resource"] = flattenAwsCloudTrailEventSelectorDataResource(raw.DataResources)

		eventSelectors = append(eventSelectors, item)
	}

	return eventSelectors
}

func flattenAwsCloudTrailEventSelectorDataResource(configured []*cloudtrail.DataResource) []map[string]interface{} {
	dataResources := make([]map[string]interface{}, 0, len(configured))

	for _, raw := range configured {
		item := make(map[string]interface{})
		item["type"] = aws.StringValue(raw.Type)
		item["values"] = flattenStringList(raw.Values)

		dataResources = append(dataResources, item)
	}

	return dataResources
}

func cloudTrailSetAdvancedEventSelectors(conn *cloudtrail.CloudTrail, d *schema.ResourceData) error {
	input := &cloudtrail.PutEventSelectorsInput{
		TrailName: aws.String(d.Id()),
	}

	advancedEventSelectors := expandAwsCloudTrailAdvancedEventSelector(d.Get("advanced_event_selector").([]interface{}))

	input.AdvancedEventSelectors = advancedEventSelectors

	if err := input.Validate(); err != nil {
		return fmt.Errorf("Error validate CloudTrail (%s): %s", d.Id(), err)
	}

	_, err := conn.PutEventSelectors(input)
	if err != nil {
		return fmt.Errorf("Error set advanced event selector on CloudTrail (%s): %s", d.Id(), err)
	}

	return nil
}

func expandAwsCloudTrailAdvancedEventSelector(configured []interface{}) []*cloudtrail.AdvancedEventSelector {
	advancedEventSelectors := make([]*cloudtrail.AdvancedEventSelector, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})
		fieldSelectors := expandAwsCloudTrailAdvancedEventSelectorFieldSelector(data["field_selector"].(*schema.Set))

		aes := &cloudtrail.AdvancedEventSelector{
			Name:           aws.String(data["name"].(string)),
			FieldSelectors: fieldSelectors,
		}

		advancedEventSelectors = append(advancedEventSelectors, aes)

	}

	return advancedEventSelectors

}

func expandAwsCloudTrailAdvancedEventSelectorFieldSelector(configured *schema.Set) []*cloudtrail.AdvancedFieldSelector {
	fieldSelectors := make([]*cloudtrail.AdvancedFieldSelector, 0, configured.Len())

	for _, raw := range configured.List() {
		data := raw.(map[string]interface{})
		fieldSelector := &cloudtrail.AdvancedFieldSelector{
			Field: aws.String(data["field"].(string)),
		}

		if v, ok := data["equals"]; ok && len(v.([]interface{})) > 0 {
			equals := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				equals[i] = aws.String(str)
			}
			fieldSelector.Equals = equals
		}

		if v, ok := data["not_equals"]; ok && len(v.([]interface{})) > 0 {
			notEquals := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				notEquals[i] = aws.String(str)
			}
			fieldSelector.NotEquals = notEquals
		}

		if v, ok := data["starts_with"]; ok && len(v.([]interface{})) > 0 {
			startsWith := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				startsWith[i] = aws.String(str)
			}
			fieldSelector.StartsWith = startsWith
		}

		if v, ok := data["not_starts_with"]; ok && len(v.([]interface{})) > 0 {
			notStartsWith := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				notStartsWith[i] = aws.String(str)
			}
			fieldSelector.NotStartsWith = notStartsWith
		}

		if v, ok := data["ends_with"]; ok && len(v.([]interface{})) > 0 {
			endsWith := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				endsWith[i] = aws.String(str)
			}
			fieldSelector.EndsWith = endsWith
		}

		if v, ok := data["not_ends_with"]; ok && len(v.([]interface{})) > 0 {
			notEndsWith := make([]*string, len(v.([]interface{})))
			for i, vv := range v.([]interface{}) {
				str := vv.(string)
				notEndsWith[i] = aws.String(str)
			}
			fieldSelector.NotEndsWith = notEndsWith
		}

		fieldSelectors = append(fieldSelectors, fieldSelector)
	}

	return fieldSelectors
}

func flattenAwsCloudTrailAdvancedEventSelector(configured []*cloudtrail.AdvancedEventSelector) []map[string]interface{} {
	advancedEventSelectors := make([]map[string]interface{}, 0, len(configured))

	for _, raw := range configured {
		item := make(map[string]interface{})
		item["name"] = aws.StringValue(raw.Name)
		item["field_selector"] = flattenAwsCloudTrailAdvancedEventSelectorFieldSelector(raw.FieldSelectors)

		advancedEventSelectors = append(advancedEventSelectors, item)
	}

	return advancedEventSelectors
}

func flattenAwsCloudTrailAdvancedEventSelectorFieldSelector(configured []*cloudtrail.AdvancedFieldSelector) []map[string]interface{} {
	fieldSelectors := make([]map[string]interface{}, 0, len(configured))

	for _, raw := range configured {
		item := make(map[string]interface{})
		item["field"] = aws.StringValue(raw.Field)
		if raw.Equals != nil {
			item["equals"] = flattenStringList(raw.Equals)
		}
		if raw.NotEquals != nil {
			item["not_equals"] = flattenStringList(raw.NotEquals)
		}
		if raw.StartsWith != nil {
			item["starts_with"] = flattenStringList(raw.StartsWith)
		}
		if raw.NotStartsWith != nil {
			item["not_starts_with"] = flattenStringList(raw.NotStartsWith)
		}
		if raw.EndsWith != nil {
			item["ends_with"] = flattenStringList(raw.EndsWith)
		}
		if raw.NotEndsWith != nil {
			item["not_ends_with"] = flattenStringList(raw.NotEndsWith)
		}

		fieldSelectors = append(fieldSelectors, item)
	}

	return fieldSelectors
}

func cloudTrailSetInsightSelectors(conn *cloudtrail.CloudTrail, d *schema.ResourceData) error {
	input := &cloudtrail.PutInsightSelectorsInput{
		TrailName: aws.String(d.Id()),
	}

	insightSelector := expandAwsCloudTrailInsightSelector(d.Get("insight_selector").([]interface{}))
	input.InsightSelectors = insightSelector

	if err := input.Validate(); err != nil {
		return fmt.Errorf("Error validate CloudTrail (%s): %s", d.Id(), err)
	}

	_, err := conn.PutInsightSelectors(input)
	if err != nil {
		return fmt.Errorf("Error set insight selector on CloudTrail (%s): %s", d.Id(), err)
	}

	return nil
}

func expandAwsCloudTrailInsightSelector(configured []interface{}) []*cloudtrail.InsightSelector {
	insightSelectors := make([]*cloudtrail.InsightSelector, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		is := &cloudtrail.InsightSelector{
			InsightType: aws.String(data["insight_type"].(string)),
		}
		insightSelectors = append(insightSelectors, is)
	}

	return insightSelectors
}

func flattenAwsCloudTrailInsightSelector(configured []*cloudtrail.InsightSelector) []map[string]interface{} {
	insightSelectors := make([]map[string]interface{}, 0, len(configured))

	for _, raw := range configured {
		item := make(map[string]interface{})
		item["insight_type"] = aws.StringValue(raw.InsightType)

		insightSelectors = append(insightSelectors, item)
	}

	return insightSelectors
}
