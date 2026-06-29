# cardModel

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
    cardModel:
      type: object
      properties:
        card_token:
          type: string
          description: |-
            Max length:255
            Токен карты для проведения платежных операций с картой
        card_pan:
          type: string
          description: |-
            Max length:16
            Маскированный номер карты
        user_phone:
          type: string
          description: >-
            Max length:12

            Телефон держателя карты в формате 998XXXXXXXXX (поддерживается
            только для карт Uzcard и Humo)
        pinfl:
          type: string
          description: >-
            ПИНФЛ держателя карты (возвращается в случае передачи в запросе на
            создание привязки для карт Uzcard и Humo)

            Max length:14
        expiry:
          type: string
          description: Срок действия карты (YYMM)
        holder_name:
          type: string
          description: Имя и фамилия владельца карты
        redirect_url:
          type: string
          description: >-
            URL для перенаправления после успешного добавления карты
            пользователем
        redirect_decline_url:
          type: string
          description: >-
            URL для перенаправления при неуспешном добавлении карты или отмены
            пользователем
        session_id:
          type: string
          description: >-
            Необходимо сохранить данное значение, оно возвращается в
            callback-запросе в поле payer_id. Также можно проверить состояние
            через метод проверки привязанной карты
        form_url:
          type: string
          description: >-
            URL формы добавления карты (необходимо перенаправить пользователя,
            либо открыть webView)
      x-apidog-orders:
        - card_token
        - card_pan
        - user_phone
        - pinfl
        - expiry
        - holder_name
        - redirect_url
        - redirect_decline_url
        - session_id
        - form_url
      required:
        - card_token
        - card_pan
        - expiry
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
