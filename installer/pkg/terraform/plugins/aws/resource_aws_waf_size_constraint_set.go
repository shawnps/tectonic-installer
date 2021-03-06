package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafSizeConstraintSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafSizeConstraintSetCreate,
		Read:   resourceAwsWafSizeConstraintSetRead,
		Update: resourceAwsWafSizeConstraintSetUpdate,
		Delete: resourceAwsWafSizeConstraintSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size_constraints": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field_to_match": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"comparison_operator": {
							Type:     schema.TypeString,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"text_transformation": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafSizeConstraintSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating SizeConstraintSet: %s", d.Get("name").(string))

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateSizeConstraintSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateSizeConstraintSet(params)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SizeConstraintSet: {{err}}", err)
	}
	resp := out.(*waf.CreateSizeConstraintSetOutput)

	d.SetId(*resp.SizeConstraintSet.SizeConstraintSetId)

	return resourceAwsWafSizeConstraintSetUpdate(d, meta)
}

func resourceAwsWafSizeConstraintSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading SizeConstraintSet: %s", d.Get("name").(string))
	params := &waf.GetSizeConstraintSetInput{
		SizeConstraintSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetSizeConstraintSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.SizeConstraintSet.Name)

	return nil
}

func resourceAwsWafSizeConstraintSetUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating SizeConstraintSet: %s", d.Get("name").(string))
	err := updateSizeConstraintSetResource(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}
	return resourceAwsWafSizeConstraintSetRead(d, meta)
}

func resourceAwsWafSizeConstraintSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Deleting SizeConstraintSet: %s", d.Get("name").(string))
	err := updateSizeConstraintSetResource(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
	}

	wr := newWafRetryer(conn, "global")
	_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(d.Id()),
		}
		return conn.DeleteSizeConstraintSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
	}

	return nil
}

func updateSizeConstraintSetResource(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(d.Id()),
		}

		sizeConstraints := d.Get("size_constraints").(*schema.Set)
		for _, sizeConstraint := range sizeConstraints.List() {
			sc := sizeConstraint.(map[string]interface{})
			sizeConstraintUpdate := &waf.SizeConstraintSetUpdate{
				Action: aws.String(ChangeAction),
				SizeConstraint: &waf.SizeConstraint{
					FieldToMatch:       expandFieldToMatch(sc["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
					ComparisonOperator: aws.String(sc["comparison_operator"].(string)),
					Size:               aws.Int64(int64(sc["size"].(int))),
					TextTransformation: aws.String(sc["text_transformation"].(string)),
				},
			}
			req.Updates = append(req.Updates, sizeConstraintUpdate)
		}

		return conn.UpdateSizeConstraintSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}

	return nil
}
