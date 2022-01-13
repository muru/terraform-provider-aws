package s3

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceBucketLifecycleConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketLifecycleConfigurationCreate,
		ReadContext:   resourceBucketLifecycleConfigurationRead,
		UpdateContext: resourceBucketLifecycleConfigurationUpdate,
		DeleteContext: resourceBucketLifecycleConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 63),
			},

			"expected_bucket_owner": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidAccountID,
			},

			"rule": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"abort_incomplete_multipart_upload": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"days_after_initiation": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"expiration": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: verify.ValidUTCTimestamp,
									},
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      0, // API returns 0
										ValidateFunc: validation.IntAtLeast(1),
									},
									"expired_object_delete_marker": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true, // API returns false
									},
								},
							},
						},
						"filter": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"and": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"object_size_greater_than": {
													Type:         schema.TypeInt,
													Optional:     true,
													ValidateFunc: validation.IntAtLeast(0),
												},
												"object_size_less_than": {
													Type:         schema.TypeInt,
													Optional:     true,
													ValidateFunc: validation.IntAtLeast(1),
												},
												"prefix": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"tags": tftags.TagsSchema(),
											},
										},
									},
									"object_size_greater_than": {
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      0, // API returns 0
										ValidateFunc: validation.IntAtLeast(0),
									},
									"object_size_less_than": {
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      0, // API returns 0
										ValidateFunc: validation.IntAtLeast(1),
									},
									"prefix": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"tag": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"key": {
													Type:     schema.TypeString,
													Required: true,
												},
												"value": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},

						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 255),
						},

						"noncurrent_version_expiration": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"newer_noncurrent_versions": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(1),
									},
									"noncurrent_days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(1),
									},
								},
							},
						},
						"noncurrent_version_transition": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"newer_noncurrent_versions": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(1),
									},
									"noncurrent_days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(0),
									},
									"storage_class": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(s3.TransitionStorageClass_Values(), false),
									},
								},
							},
						},

						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"status": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								LifecycleRuleStatusDisabled,
								LifecycleRuleStatusEnabled,
							}, false),
						},

						"transition": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: verify.ValidUTCTimestamp,
									},
									"days": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntAtLeast(0),
									},
									"storage_class": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(s3.TransitionStorageClass_Values(), false),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceBucketLifecycleConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	bucket := d.Get("bucket").(string)

	rules, err := ExpandLifecycleRules(d.Get("rule").(*schema.Set).List())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating S3 Lifecycle Configuration for bucket (%s): %w", bucket, err))
	}

	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}

	if v, ok := d.GetOk("expected_bucket_owner"); ok {
		input.ExpectedBucketOwner = aws.String(v.(string))
	}

	_, err = verify.RetryOnAWSCode(s3.ErrCodeNoSuchBucket, func() (interface{}, error) {
		return conn.PutBucketLifecycleConfigurationWithContext(ctx, input)
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating S3 lifecycle configuration for bucket (%s): %w", bucket, err))
	}

	d.SetId(bucket)

	return resourceBucketLifecycleConfigurationRead(ctx, d, meta)
}

func resourceBucketLifecycleConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	input := &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(d.Id()),
	}

	output, err := verify.RetryOnAWSCode(ErrCodeNoSuchLifecycleConfiguration, func() (interface{}, error) {
		return conn.GetBucketLifecycleConfigurationWithContext(ctx, input)
	})

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, ErrCodeNoSuchLifecycleConfiguration, s3.ErrCodeNoSuchBucket) {
		log.Printf("[WARN] S3 Bucket Lifecycle Configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error getting S3 Bucket Lifecycle Configuration for bucket (%s): %w", d.Id(), err))
	}

	lifecycleConfig, ok := output.(*s3.GetBucketLifecycleConfigurationOutput)

	if !ok || lifecycleConfig == nil {
		return diag.FromErr(fmt.Errorf("error reading S3 Bucket Lifecycle Configuration for bucket (%s): empty output", d.Id()))
	}

	d.Set("bucket", d.Id())
	if err := d.Set("rule", FlattenLifecycleRules(lifecycleConfig.Rules)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting rule: %w", err))
	}

	return nil
}

func resourceBucketLifecycleConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	rules, err := ExpandLifecycleRules(d.Get("rule").(*schema.Set).List())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating S3 Bucket Lifecycle Configuration rule: %w", err))
	}

	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(d.Id()),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}

	_, err = verify.RetryOnAWSCode(ErrCodeNoSuchLifecycleConfiguration, func() (interface{}, error) {
		return conn.PutBucketLifecycleConfigurationWithContext(ctx, input)
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating S3 lifecycle configuration for bucket (%s): %w", d.Id(), err))
	}

	if err := waitForLifecycleConfigurationRulesStatus(ctx, conn, d.Id(), rules); err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for S3 lifecycle configuration for bucket (%s) to reach expected rules status after update: %w", d.Id(), err))
	}

	return resourceBucketLifecycleConfigurationRead(ctx, d, meta)
}

func resourceBucketLifecycleConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).S3Conn

	input := &s3.DeleteBucketLifecycleInput{
		Bucket: aws.String(d.Id()),
	}

	_, err := conn.DeleteBucketLifecycleWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, ErrCodeNoSuchLifecycleConfiguration, s3.ErrCodeNoSuchBucket) {
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting S3 bucket lifecycle configuration for bucket (%s): %w", d.Id(), err))
	}

	return nil
}
