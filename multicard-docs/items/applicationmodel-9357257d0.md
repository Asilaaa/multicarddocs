# applicationModel

## OpenAPI Specification

```yaml
openapi: 3.0.1
info:
  title: ''
  description: ''
  version: 1.0.0
paths: {}
components:
  schemas:
    applicationModel:
      type: object
      properties:
        id:
          type: integer
        application_id:
          type: string
        wallet_sum:
          type: integer
          nullable: true
        wallet_sender_account:
          type: string
          nullable: true
        wallet_overdraft:
          type: integer
        wallet_contract_num:
          type: string
          nullable: true
        otp_required:
          type: integer
        otp_gateway:
          type: string
          nullable: true
        sms_nickname:
          type: string
          nullable: true
      x-apidog-orders:
        - id
        - application_id
        - wallet_sum
        - wallet_sender_account
        - wallet_overdraft
        - wallet_contract_num
        - otp_required
        - otp_gateway
        - sms_nickname
      required:
        - id
        - application_id
        - wallet_sum
        - otp_required
        - wallet_contract_num
        - wallet_overdraft
        - wallet_sender_account
        - sms_nickname
        - otp_gateway
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
