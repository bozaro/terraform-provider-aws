package finder

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscalingplans"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func ScalingPlanByNameAndVersion(conn *autoscalingplans.AutoScalingPlans, scalingPlanName string, scalingPlanVersion int) (*autoscalingplans.ScalingPlan, error) {
	input := &autoscalingplans.DescribeScalingPlansInput{
		ScalingPlanNames:   aws.StringSlice([]string{scalingPlanName}),
		ScalingPlanVersion: aws.Int64(int64(scalingPlanVersion)),
	}

	output, err := conn.DescribeScalingPlans(input)

	if tfawserr.ErrCodeEquals(err, autoscalingplans.ErrCodeObjectNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.ScalingPlans) == 0 || output.ScalingPlans[0] == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.ScalingPlans[0], nil
}
