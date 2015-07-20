# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [2.0.0] - unreleased
- Mailer has been removed. It has been replaced by Dialer and Sender.
- `CreateFile` has been removed and `OpenFile` has been renamed into `NewFile`.
- `File` fields changed. The `File.Header` field replaces `File.MimeType`
and `File.ContentID`.
- `Message.GetBodyWriter` has been removed. Use `Message.AddAlternativeWriter`
instead.
- `Message.Export` has been removed. `Message.WriteTo` can be used instead.
- The `Bcc` header field is no longer sent. It is far more simpler and
efficient: the same message is sent to all recipients instead of sending a
different email to each Bcc address.
- LoginAuth has been removed. `NewPlainDialer` now implements the LOGIN
authentication mechanism when needed.
- Go 1.2 is now required instead of Go 1.3. No external dependency are used when
using Go 1.5.
