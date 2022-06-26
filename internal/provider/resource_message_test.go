package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceMessage(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceMessage,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("smtp_message.test", "body", "Boom"),
				),
			},
		},
	})
}

const testAccResourceMessage = `
provider "smtp" {
  plain_auth {}
}

resource "smtp_message" "test" {
  subject = "Hello World!"
  body = "Boom"
  to = ["devnull@spacelift.io"]
}
`
