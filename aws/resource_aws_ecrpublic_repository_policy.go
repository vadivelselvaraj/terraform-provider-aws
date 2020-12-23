package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecrpublic"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsEcrPublicRepositoryPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrPublicRepositoryPolicyCreate,
		Read:   resourceAwsEcrPublicRepositoryPolicyRead,
		Update: resourceAwsEcrPublicRepositoryPolicyUpdate,
		Delete: resourceAwsEcrPublicRepositoryPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"force": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"policy_text": {
				Type:     schema.TypeString,
				Required: true,
			},
			"registry_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"repository_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsEcrPublicRepositoryPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	registryID := d.Get("registry_id").(string)
	repositoryName := d.Get("repository_name").(string)

	input := ecrpublic.SetRepositoryPolicyInput{
		Force:          aws.Bool(d.Get("force").(bool)),
		PolicyText:     aws.String(d.Get("policy_text").(string)),
		RegistryId:     aws.String(registryID),
		RepositoryName: aws.String(repositoryName),
	}

	log.Printf("[DEBUG] Creating ECR Public repository policy: %#v", input)
	_, err := conn.SetRepositoryPolicy(&input)
	if err != nil {
		return fmt.Errorf("error creating ECR Public repository policy: %s", err)
	}

	log.Printf("[DEBUG] ECR Public repository created: %s", d.Get("repository_name").(string))

	d.SetId(encodeRepositoryPolicyID(registryID, repositoryName))

	return resourceAwsEcrPublicRepositoryRead(d, meta)
}

func resourceAwsEcrPublicRepositoryPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	registryID, repositoryName, err := decodeRepositoryPolicyID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading ECR Public repository policy %s", d.Id())
	var out *ecrpublic.GetRepositoryPolicyOutput
	input := &ecrpublic.GetRepositoryPolicyInput{
		RegistryId:     aws.String(registryID),
		RepositoryName: aws.String(repositoryName),
	}

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		out, err = conn.GetRepositoryPolicy(input)
		if d.IsNewResource() && isAWSErr(err, ecrpublic.ErrCodeRepositoryPolicyNotFoundException, "") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if isResourceTimeoutError(err) {
		out, err = conn.GetRepositoryPolicy(input)
	}

	if isAWSErr(err, ecrpublic.ErrCodeRepositoryPolicyNotFoundException, "") {
		log.Printf("[WARN] ECR Public Repository policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading ECR Public repository policy: %s", err)
	}

	d.Set("repository_name", out.RepositoryName)
	d.Set("registry_id", out.RegistryId)
	d.Set("policy_text", out.PolicyText)

	return nil
}

func resourceAwsEcrPublicRepositoryPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	registryID, repositoryName, err := decodeRepositoryPolicyID(d.Id())
	if err != nil {
		return err
	}

	_, err = conn.DeleteRepositoryPolicy(&ecrpublic.DeleteRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
		RegistryId:     aws.String(registryID),
	})
	if err != nil {
		if isAWSErr(err, ecrpublic.ErrCodeRepositoryPolicyNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting ECR Public repository policy: %s", err)
	}

	log.Printf("[DEBUG] Waiting for ECR Public Repository policy %q to be deleted", d.Id())

	input := &ecrpublic.GetRepositoryPolicyInput{
		RegistryId:     aws.String(registryID),
		RepositoryName: aws.String(repositoryName),
	}

	_, err = conn.GetRepositoryPolicy(input)

	if isAWSErr(err, ecrpublic.ErrCodeRepositoryPolicyNotFoundException, "") {
		return nil
	}

	return nil
}

func resourceAwsEcrPublicRepositoryPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	registryID, repositoryName, err := decodeRepositoryPolicyID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("policy_text") {
		input := ecrpublic.SetRepositoryPolicyInput{
			Force:          aws.Bool(d.Get("force").(bool)),
			PolicyText:     aws.String(d.Get("policy_text").(string)),
			RegistryId:     aws.String(registryID),
			RepositoryName: aws.String(repositoryName),
		}

		log.Printf("[DEBUG] Creating ECR Public repository policy: %#v", input)
		_, err := conn.SetRepositoryPolicy(&input)
		if err != nil {
			return fmt.Errorf("error creating ECR Public repository policy: %s", err)
		}
	}

	return resourceAwsEcrPublicRepositoryRead(d, meta)
}

func encodeRepositoryPolicyID(registryId, repositoryId string) string {
	return fmt.Sprintf("%s|%s", registryId, repositoryId)
}

func decodeRepositoryPolicyID(id string) (string, string, error) {
	idParts := strings.Split(id, "|")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format RegistryId|RepositoryId, received: %s", id)
	}
	return idParts[0], idParts[1], nil
}
