resource "smtp_message" "test" {
  subject = "Hello"
  body    = "World!"
  to      = ["devnull@spacelift.io"]
}
