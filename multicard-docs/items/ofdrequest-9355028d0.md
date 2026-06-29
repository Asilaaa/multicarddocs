# ofdRequest

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
    ofdRequest:
      type: array
      items:
        type: object
        properties:
          qty:
            type: integer
            description: Количество единиц товара/услуги
          vat:
            type: integer
            description: НДС (%)
          price:
            type: integer
            description: Стоимость единицы товара в тийинах
          mxik:
            type: string
            description: ИКПУ из справочника tasnif.soliq.uz
          total:
            type: integer
            description: Общая сумма товаров с учетом количества без учета скидок в тийинах
          package_code:
            type: string
            description: Код упаковки из справочника tasnif.soliq.uz
          name:
            type: string
            description: Наименование товара/услуги
          tin:
            type: string
            description: ИНН компании
          mark:
            type: array
            items:
              type: string
              description: Код маркировки
            description: >-
              Массив с кодами маркировок каждой единицы товара. Обязателен для
              маркировочных товаров
        x-apidog-orders:
          - qty
          - vat
          - price
          - mxik
          - total
          - package_code
          - name
          - tin
          - mark
        required:
          - qty
          - price
          - mxik
          - package_code
          - name
          - total
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
