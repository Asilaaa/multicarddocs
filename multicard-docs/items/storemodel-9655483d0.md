# storeModel

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
    billingFieldsModel:
      type: array
      items:
        type: object
        properties:
          type:
            type: string
            description: Формат поля
            enum:
              - string
              - int
              - phone
              - tree
              - hidden
              - complex
              - select
            x-apidog-enum:
              - value: string
                name: ''
                description: ''
              - value: int
                name: ''
                description: ''
              - value: phone
                name: ''
                description: ''
              - value: tree
                name: ''
                description: ''
              - value: hidden
                name: ''
                description: ''
              - value: complex
                name: ''
                description: ''
              - value: select
                name: ''
                description: ''
          name:
            type: string
            description: Описание поля
          value:
            type: string
          key:
            type: string
            description: Ключ
          suggested:
            type: boolean
            description: Рекоммендуемая сумма оплаты
            nullable: true
        x-apidog-orders:
          - type
          - name
          - value
          - key
          - suggested
        required:
          - type
          - value
          - name
          - key
          - suggested
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
    storeModel:
      type: object
      properties:
        id:
          type: integer
        uuid:
          type: string
          format: uuid
        category_id:
          type: integer
          nullable: true
        note:
          type: string
          nullable: true
        logo:
          type: string
          description: Логотип
          nullable: true
        color:
          type: string
          nullable: true
        view_fields:
          anyOf:
            - $ref: '#/components/schemas/billingFieldsModel'
            - type: 'null'
        tax_registration:
          type: integer
          enum:
            - 0
            - 1
            - 2
            - 3
          x-apidog-enum:
            - value: 0
              name: ''
              description: Фискализация отключена
            - value: 1
              name: ''
              description: Фискализация включена
            - value: 2
              name: ''
              description: Фискализируется только комиссия сверху
            - value: 3
              name: ''
              description: Фискализируется только комиссия снизу
          description: Флаг фискализации
        tax_mxik:
          type: string
          description: ИКПУ по умолчанию из каталога tasnif.soliq.uz
          nullable: true
        tax_package_code:
          type: string
          description: Код упаковки от ИКПУ по умолчанию из каталога tasnif.soliq.uz
          nullable: true
        tax_commission_recipient_tin:
          type: string
          description: ИНН комитента для фискализации. Если null, то берется ИНН мерчанта
          nullable: true
        tg_chat_id:
          type: string
          description: ID телеграм группы для отправки уведомлений о платежах
          nullable: true
        qr_url:
          type: string
          description: Ссылка на QR-код для приема платежей по данной кассе
        bg_img:
          type: string
          description: Фоновая картинка для страницы чекаута
        title:
          type: string
          description: Наименование кассы
        merchant:
          $ref: '#/components/schemas/merchantModel'
          description: Информация о клиенте (мерчанте)
        contract:
          type: object
          properties:
            id:
              type: integer
            num:
              type: string
              description: Номер договора
            date:
              type: string
              format: date
              description: Дата договора
            service:
              type: string
            fee:
              type: object
              properties:
                up:
                  type: string
                down:
                  type: string
              x-apidog-orders:
                - up
                - down
              required:
                - up
                - down
              description: Комиссия по договору
            edm_document_id:
              type: string
              description: Идентификатор документа в системе электронного документооборота
              nullable: true
            edm_status:
              type: string
              description: >-
                Статус подписания документа в системе электронного
                документооборота
              nullable: true
          x-apidog-orders:
            - id
            - num
            - date
            - service
            - fee
            - edm_document_id
            - edm_status
          description: Информация о контракте с мерчантом
          required:
            - id
            - service
            - date
            - num
            - fee
            - edm_document_id
            - edm_status
          nullable: true
        merchant_account:
          $ref: '#/components/schemas/merchantAccount'
      x-apidog-orders:
        - id
        - uuid
        - category_id
        - note
        - logo
        - color
        - view_fields
        - tax_registration
        - tax_mxik
        - tax_package_code
        - tax_commission_recipient_tin
        - tg_chat_id
        - qr_url
        - bg_img
        - title
        - merchant
        - contract
        - merchant_account
      required:
        - id
        - uuid
        - category_id
        - tax_registration
        - view_fields
        - color
        - logo
        - note
        - bg_img
        - qr_url
        - tg_chat_id
        - tax_commission_recipient_tin
        - tax_package_code
        - tax_mxik
        - merchant
        - title
        - merchant_account
        - contract
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
