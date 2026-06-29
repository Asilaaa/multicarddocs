# clearingModel

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
    clearingModel:
      type: object
      properties:
        id:
          type: integer
        merchant:
          $ref: '#/components/schemas/merchantModel'
          description: Информация о клиенте (мерчанте)
        recipient_info: &ref_0
          $ref: '#/components/schemas/merchantAccount'
        sender_info: *ref_0
        purpose_code:
          type: string
          description: Код назначения платежа
        amount:
          type: integer
          description: Сумма платежа в тийинах
        details:
          type: string
          description: Детали платежа
        status:
          type: string
          enum:
            - new
            - sent
            - done
            - repeat
            - postponed
            - blocked
            - revert
            - canceled
          x-apidog-enum:
            - value: new
              name: ''
              description: ''
            - value: sent
              name: ''
              description: ''
            - value: done
              name: ''
              description: ''
            - value: repeat
              name: ''
              description: ''
            - value: postponed
              name: ''
              description: ''
            - value: blocked
              name: ''
              description: ''
            - value: revert
              name: ''
              description: ''
            - value: canceled
              name: ''
              description: ''
          description: Статус
        payment_time:
          type: string
          description: Время проведения платежа
        added_on:
          type: string
          description: Дата создания записи
        updated_on:
          type: string
          description: Дата изменения записи
        receipt_url:
          type: string
          description: URL на банковскую квитанцию
      x-apidog-orders:
        - id
        - merchant
        - recipient_info
        - sender_info
        - purpose_code
        - amount
        - details
        - status
        - payment_time
        - added_on
        - updated_on
        - receipt_url
      required:
        - id
        - merchant
        - recipient_info
        - sender_info
        - purpose_code
        - status
        - details
        - amount
        - updated_on
        - added_on
        - payment_time
        - receipt_url
      x-apidog-folder: ''
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
    merchantModel:
      type: object
      properties:
        id:
          type: integer
          description: ID клиента (мерчанта) в Multicard
        name:
          type: string
          description: Наименование клиента (мерчанта)
        tin:
          type: string
          description: ИНН мерчанта
        contract_id:
          type: string
          description: Данные о договоре
          nullable: true
        bank_account:
          type: string
          description: Транзитный счет для расчетов
          nullable: true
      x-apidog-orders:
        - id
        - name
        - tin
        - contract_id
        - bank_account
      required:
        - id
        - name
        - tin
        - bank_account
        - contract_id
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
