# merchantAccount

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
    merchantAccount:
      type: object
      properties:
        id:
          type: integer
          description: ID банковских реквизитов
        tin:
          type: string
          description: ИНН или ПИНФЛ
        uuid:
          type: string
          format: uuid
          description: UUID банковских реквизитов
        official_name:
          type: string
          description: Наименование
        mfo:
          type: string
          description: МФО
        account_no:
          type: string
          description: Номер счета
        address:
          type: string
          description: Юридический адрес
          nullable: true
        director:
          type: string
          description: ФИО директора
          nullable: true
        director_pinfl:
          type: string
          description: ПИНФЛ директора
          nullable: true
        vat_payer:
          type: boolean
          description: Является ли плательщиков НДС
          nullable: true
        is_commitent:
          type: boolean
          description: Является ли комитентом Multicard
          nullable: true
        active:
          type: boolean
          description: >-
            Флаг активности. Если отключен, значит возмещение не будет
            отправлено
      x-apidog-orders:
        - id
        - uuid
        - tin
        - official_name
        - mfo
        - account_no
        - address
        - director
        - director_pinfl
        - vat_payer
        - is_commitent
        - active
      required:
        - id
        - uuid
        - tin
        - mfo
        - official_name
        - vat_payer
        - director_pinfl
        - director
        - address
        - account_no
        - active
        - is_commitent
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
