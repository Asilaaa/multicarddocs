# taxReceiptModel

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
    taxReceiptModel:
      type: object
      properties:
        receipt:
          type: object
          properties: {}
          x-apidog-orders: []
          description: Объект чека, отправленный в ГНК
          nullable: true
        f_num:
          type: string
          format: uuid
          description: Фискальный признак
        fm_num:
          type: string
          description: Фискальный терминал
        qr_url:
          type: string
          description: URL на фискальный чек
          nullable: true
        is_refund:
          type: boolean
          description: Является ли чеком возврата
      x-apidog-orders:
        - receipt
        - f_num
        - fm_num
        - qr_url
        - is_refund
      required:
        - receipt
        - f_num
        - fm_num
        - is_refund
        - qr_url
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
