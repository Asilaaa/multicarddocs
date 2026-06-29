# Callback (webhooks)

## OpenAPI Specification

```yaml
openapi: 3.0.1
info:
  title: ''
  description: ''
  version: 1.0.0
paths:
  /:
    post:
      summary: Callback (webhooks)
      deprecated: false
      description: >-
        Возможно настройка мерчанта таким образом, что callback-запрос будет
        отправлятся каждый раз при смене статуса платежной транзакции.


        Перечень статусов (отправляемых запросов):


        - draft
            
        - progress
            
        - success
            
        - error
            
        - revert
            

        Запрос отправляется со следующего IP: 195.158.26.90. При получении
        запроса система мерчанта должна проверить поле sign, либо установить
        ограничение на прием запросов только с этого IP-адреса.


        > sign формируется как sha1-хеш от строки (без скобок):  

        {uuid}{invoice_id}{amount}{secret} 
          

        Для успешного ответа на указанный запрос необходимо ответить HTTP
        STATUS=2xx. В противном случае коллбэк-запрос будет повторен 5 раз.
      tags:
        - Оплата на платежной странице Multicard
      parameters: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                uuid:
                  type: string
                  description: Уникальный идентификатор транзакции в Multicard
                amount:
                  type: integer
                  description: Сумма платежа в тийинах
                invoice_id:
                  type: string
                  description: >-
                    Max length:255

                    Любой идентификатор заказа в системе Партнера. Будет
                    возвращен в callback-запросе. Также по нему можно искать
                    платежи в кабинете мерчанта
                status:
                  type: string
                  description: |-
                    Allowed values (6)
                    draft,
                    progress,
                    success,
                    error,
                    revert,
                    hold
                     hide all
                billing_id:
                  type: string
                  description: Уникальный ID транзакции в системе Партнера
                payment_time:
                  type: string
                  description: Время списания средств с карты клиента
                refund_time:
                  type: 'null'
                  description: Время зачисления средств обратно на карту клиента
                phone:
                  type: string
                  description: >-
                    Max length:12

                    Телефон держателя карты в формате 998XXXXXXXXX
                    (поддерживается только для карт Uzcard и Humo)
                card_pan:
                  type: string
                  description: |-
                    Max length:16
                    Маскированный номер карты
                ps:
                  type: string
                  description: |-
                    Allowed values (5)
                    uzcard,
                    humo,
                    visa,
                    mastercard,
                    unionpay
                    hide all
                    Платежная система
                card_token:
                  type: string
                  description: |-
                    Max length:255
                    Токен карты для проведения платежных операций с картой
                receipt_url:
                  type: 'null'
                  description: Ссылка на фискальный чек
                sign:
                  type: string
                  description: |+
                    SHA1 от строки: {uuid}{invoice_id}{amount}{secret}

              required:
                - uuid
                - amount
                - invoice_id
                - status
                - billing_id
                - payment_time
                - refund_time
                - phone
                - card_pan
                - ps
                - card_token
                - receipt_url
                - sign
              x-apidog-orders:
                - uuid
                - amount
                - invoice_id
                - status
                - billing_id
                - payment_time
                - refund_time
                - phone
                - card_pan
                - ps
                - card_token
                - receipt_url
                - sign
            example:
              uuid: e60d8ebc-b9fe-11ef-b159-005056b4367d
              amount: 200000
              invoice_id: test
              status: progress
              billing_id: '20241214242009869794410864028760'
              payment_time: '2024-12-14 14:36:31'
              refund_time: null
              phone: '998930601725'
              card_pan: 860030******5959
              ps: uzcard
              card_token: 6225f3c93f7a880142782fa4
              receipt_url: null
              sign: 5fe1289fbe9cf1834de72789f6629e5f23184974
      responses:
        '200':
          description: ''
          content:
            application/json:
              schema:
                type: object
                properties: {}
                x-apidog-orders: []
          headers: {}
          x-apidog-name: Success
      security: []
      x-apidog-folder: Оплата на платежной странице Multicard
      x-apidog-status: released
      x-run-in-apidog: https://app.apidog.com/web/project/1022226/apis/api-19729301-run
components:
  schemas: {}
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
