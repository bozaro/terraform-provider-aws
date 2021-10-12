package ec2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccAWSEbsVolumeDataSource_basic(t *testing.T) {
	resourceName := "aws_ebs_volume.test"
	dataSourceName := "data.aws_ebs_volume.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEBSVolumeDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeIDDataSource(dataSourceName),
					resource.TestCheckResourceAttrPair(dataSourceName, "arn", resourceName, "arn"),
					resource.TestCheckResourceAttrPair(dataSourceName, "size", resourceName, "size"),
					resource.TestCheckResourceAttrPair(dataSourceName, "tags", resourceName, "tags"),
					resource.TestCheckResourceAttrPair(dataSourceName, "outpost_arn", resourceName, "outpost_arn"),
					resource.TestCheckResourceAttrPair(dataSourceName, "multi_attach_enabled", resourceName, "multi_attach_enabled"),
					resource.TestCheckResourceAttrPair(dataSourceName, "throughput", resourceName, "throughput"),
				),
			},
		},
	})
}

func TestAccAWSEbsVolumeDataSource_multipleFilters(t *testing.T) {
	resourceName := "aws_ebs_volume.test"
	dataSourceName := "data.aws_ebs_volume.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEBSVolumeWithMultipleFiltersDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeIDDataSource(dataSourceName),
					resource.TestCheckResourceAttrPair(dataSourceName, "size", resourceName, "size"),
					resource.TestCheckResourceAttr(dataSourceName, "volume_type", "gp2"),
					resource.TestCheckResourceAttrPair(dataSourceName, "tags", resourceName, "tags"),
				),
			},
		},
	})
}

func testAccCheckEBSVolumeIDDataSource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Volume data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Volume data source ID not set")
		}
		return nil
	}
}

var testAccCheckEBSVolumeDataSourceConfig = acctest.ConfigAvailableAZsNoOptIn() + `
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  type              = "gp2"
  size              = 40

  tags = {
    Name = "External Volume"
  }
}

data "aws_ebs_volume" "test" {
  most_recent = true

  filter {
    name   = "tag:Name"
    values = ["External Volume"]
  }

  filter {
    name   = "volume-type"
    values = [aws_ebs_volume.test.type]
  }
}
`

var testAccCheckEBSVolumeWithMultipleFiltersDataSourceConfig = acctest.ConfigAvailableAZsNoOptIn() + `
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  type              = "gp2"
  size              = 10

  tags = {
    Name = "External Volume 1"
  }
}

data "aws_ebs_volume" "test" {
  most_recent = true

  filter {
    name   = "tag:Name"
    values = ["External Volume 1"]
  }

  filter {
    name   = "size"
    values = [aws_ebs_volume.test.size]
  }

  filter {
    name   = "volume-type"
    values = [aws_ebs_volume.test.type]
  }
}
`
