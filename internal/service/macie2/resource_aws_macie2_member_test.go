package macie2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/envvar"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/provider"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

const (
	EnvVarMacie2PrincipalEmail             = "AWS_MACIE2_ACCOUNT_EMAIL"
	EnvVarMacie2AlternateEmail             = "AWS_MACIE2_ALTERNATE_ACCOUNT_EMAIL"
	EnvVarMacie2PrincipalEmailMessageError = "Environment variable AWS_MACIE2_ACCOUNT_EMAIL is not set. " +
		"To properly test inviting Macie member account must be provided."
	EnvVarMacie2AlternateEmailMessageError = "Environment variable AWS_MACIE2_ALTERNATE_ACCOUNT_EMAIL is not set. " +
		"To properly test inviting Macie member account must be provided."
)

func testAccAwsMacie2Member_basic(t *testing.T) {
	var providers []*schema.Provider
	var macie2Output macie2.GetMemberOutput
	resourceName := "aws_macie2_member.member"
	dataSourceAlternate := "data.aws_caller_identity.member"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2MemberDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigBasic(acctest.DefaultEmailAddress),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusCreated),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config:            testAccAwsMacieMemberConfigBasic(acctest.DefaultEmailAddress),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAwsMacie2Member_disappears(t *testing.T) {
	var providers []*schema.Provider
	var macie2Output macie2.GetMemberOutput
	resourceName := "aws_macie2_member.member"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2MemberDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigBasic(acctest.DefaultEmailAddress),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					acctest.CheckResourceDisappears(acctest.Provider, ResourceMember(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAwsMacie2Member_invite(t *testing.T) {
	var macie2Output macie2.GetMemberOutput
	var providers []*schema.Provider
	resourceName := "aws_macie2_member.member"
	dataSourceAlternate := "data.aws_caller_identity.member"
	email := conns.SkipIfEnvVarEmpty(t, EnvVarMacie2AlternateEmail, EnvVarMacie2AlternateEmailMessageError)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2InvitationAccepterDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigInvite(email, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusCreated),
					resource.TestCheckResourceAttr(resourceName, "invite", "false"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config: testAccAwsMacieMemberConfigInvite(email, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusInvited),
					resource.TestCheckResourceAttr(resourceName, "invite", "true"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config:                  testAccAwsMacieMemberConfigInvite(email, true),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"invitation_message"},
			},
		},
	})
}

func testAccAwsMacie2Member_inviteRemoved(t *testing.T) {
	var macie2Output macie2.GetMemberOutput
	var providers []*schema.Provider
	resourceName := "aws_macie2_member.member"
	dataSourceAlternate := "data.aws_caller_identity.member"
	email := conns.SkipIfEnvVarEmpty(t, EnvVarMacie2AlternateEmail, EnvVarMacie2AlternateEmailMessageError)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2InvitationAccepterDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigInvite(email, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusInvited),
					resource.TestCheckResourceAttr(resourceName, "invite", "true"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config: testAccAwsMacieMemberConfigInvite(email, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusRemoved),
					resource.TestCheckResourceAttr(resourceName, "invite", "false"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config:                  testAccAwsMacieMemberConfigInvite(email, false),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"invitation_message"},
			},
		},
	})
}

func testAccAwsMacie2Member_status(t *testing.T) {
	var macie2Output macie2.GetMemberOutput
	var providers []*schema.Provider
	resourceName := "aws_macie2_member.member"
	dataSourceAlternate := "data.aws_caller_identity.member"
	email := conns.SkipIfEnvVarEmpty(t, EnvVarMacie2AlternateEmail, EnvVarMacie2AlternateEmailMessageError)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2InvitationAccepterDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigStatus(email, macie2.MacieStatusEnabled, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusInvited),
					resource.TestCheckResourceAttr(resourceName, "invite", "true"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusEnabled),
				),
			},
			{
				Config: testAccAwsMacieMemberConfigStatus(email, macie2.MacieStatusPaused, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					resource.TestCheckResourceAttr(resourceName, "relationship_status", macie2.RelationshipStatusPaused),
					resource.TestCheckResourceAttr(resourceName, "invite", "true"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "status", macie2.MacieStatusPaused),
				),
			},
			{
				Config:                  testAccAwsMacieMemberConfigStatus(email, macie2.MacieStatusPaused, true),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"invitation_message"},
			},
		},
	})
}

func testAccAwsMacie2Member_withTags(t *testing.T) {
	var providers []*schema.Provider
	var macie2Output macie2.GetMemberOutput
	resourceName := "aws_macie2_member.member"
	dataSourceAlternate := "data.aws_caller_identity.member"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAlternateAccount(t)
		},
		ProviderFactories: acctest.FactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsMacie2MemberDestroy,
		ErrorCheck:        acctest.ErrorCheck(t, macie2.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccAwsMacieMemberConfigWithTags(acctest.DefaultEmailAddress),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsMacie2MemberExists(resourceName, &macie2Output),
					acctest.CheckResourceAttrRFC3339(resourceName, "invited_at"),
					acctest.CheckResourceAttrRFC3339(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.Key", "value"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.Key", "value"),
					acctest.CheckResourceAttrAccountID(resourceName, "administrator_account_id"),
					acctest.CheckResourceAttrAccountID(resourceName, "master_account_id"),
					resource.TestCheckResourceAttrPair(resourceName, "account_id", dataSourceAlternate, "account_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAwsMacie2MemberExists(resourceName string, macie2Session *macie2.GetMemberOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).Macie2Conn
		input := &macie2.GetMemberInput{Id: aws.String(rs.Primary.ID)}

		resp, err := conn.GetMember(input)

		if err != nil {
			return err
		}

		if resp == nil {
			return fmt.Errorf("macie Member %q does not exist", rs.Primary.ID)
		}

		*macie2Session = *resp

		return nil
	}
}

func testAccCheckAwsMacie2MemberDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).Macie2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_macie2_member" {
			continue
		}

		input := &macie2.GetMemberInput{Id: aws.String(rs.Primary.ID)}
		resp, err := conn.GetMember(input)

		if tfawserr.ErrCodeEquals(err, macie2.ErrCodeResourceNotFoundException) ||
			tfawserr.ErrMessageContains(err, macie2.ErrCodeAccessDeniedException, "Macie is not enabled") ||
			tfawserr.ErrMessageContains(err, macie2.ErrCodeConflictException, "member accounts are associated with your account") ||
			tfawserr.ErrMessageContains(err, macie2.ErrCodeValidationException, "account is not associated with your account") {
			continue
		}

		if err != nil {
			return err
		}

		if resp != nil {
			return fmt.Errorf("macie Member %q still exists", rs.Primary.ID)
		}
	}

	return nil

}

func testAccAwsMacieMemberConfigBasic(email string) string {
	return acctest.ConfigAlternateAccountProvider() + fmt.Sprintf(`
data "aws_caller_identity" "member" {
  provider = "awsalternate"
}

resource "aws_macie2_account" "admin" {}

resource "aws_macie2_member" "member" {
  account_id = data.aws_caller_identity.member.account_id
  email      = %[1]q
  depends_on = [aws_macie2_account.admin]
}
`, email)
}

func testAccAwsMacieMemberConfigWithTags(email string) string {
	return acctest.ConfigAlternateAccountProvider() + fmt.Sprintf(`
data "aws_caller_identity" "member" {
  provider = "awsalternate"
}

resource "aws_macie2_account" "admin" {}

resource "aws_macie2_member" "member" {
  account_id = data.aws_caller_identity.member.account_id
  email      = %[1]q
  tags = {
    Key = "value"
  }
  depends_on = [aws_macie2_account.admin]
}
`, email)
}

func testAccAwsMacieMemberConfigInvite(email string, invite bool) string {
	return acctest.ConfigAlternateAccountProvider() + fmt.Sprintf(`
data "aws_caller_identity" "member" {
  provider = "awsalternate"
}

data "aws_caller_identity" "admin" {}

resource "aws_macie2_account" "admin" {}

resource "aws_macie2_account" "member" {
  provider = "awsalternate"
}

resource "aws_macie2_member" "member" {
  account_id         = data.aws_caller_identity.member.account_id
  email              = %[1]q
  invite             = %[2]t
  invitation_message = "This is a message of the invitation"
  depends_on         = [aws_macie2_account.admin]
}
`, email, invite)
}

func testAccAwsMacieMemberConfigStatus(email, memberStatus string, invite bool) string {
	return acctest.ConfigAlternateAccountProvider() + fmt.Sprintf(`
data "aws_caller_identity" "member" {
  provider = "awsalternate"
}

data "aws_caller_identity" "admin" {}

resource "aws_macie2_account" "admin" {}

resource "aws_macie2_account" "member" {
  provider = "awsalternate"
}

resource "aws_macie2_member" "member" {
  account_id         = data.aws_caller_identity.member.account_id
  email              = %[1]q
  status             = %[2]q
  invite             = %[3]t
  invitation_message = "This is a message of the invitation"
  depends_on         = [aws_macie2_account.admin]
}

resource "aws_macie2_invitation_accepter" "member" {
  provider                 = "awsalternate"
  administrator_account_id = data.aws_caller_identity.admin.account_id
  depends_on               = [aws_macie2_member.member]
}
`, email, memberStatus, invite)
}
