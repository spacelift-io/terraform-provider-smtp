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

func TestAccResourceFrom(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceMessage,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("smtp_message.test", "from", "\"Hello\" <world@example.com>"),
				),
			},
		},
	})
}

const testAccResourceMessage = `
provider "smtp" {
  host = "localhost"
  username = "test"
  plain_auth {
	password = "test"
  }
}

resource "smtp_message" "test" {
  subject = "Hello World!"
  body = "Boom"
  from = "\"Hello\" <world@example.com>"
  to = ["devnull@spacelift.io"]
}
`
