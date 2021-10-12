package autoscalingplans

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscalingplans"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/service/autoscalingplans/finder"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	tfautoscalingplans "github.com/hashicorp/terraform-provider-aws/internal/service/autoscalingplans"
)

func statusScalingPlanCode(conn *autoscalingplans.AutoScalingPlans, scalingPlanName string, scalingPlanVersion int) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		scalingPlan, err := tfautoscalingplans.FindScalingPlanByNameAndVersion(conn, scalingPlanName, scalingPlanVersion)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return scalingPlan, aws.StringValue(scalingPlan.StatusCode), nil
	}
}