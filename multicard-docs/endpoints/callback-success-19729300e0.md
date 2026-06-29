# Callback (success)

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
      summary: Callback (success)
      deprecated: false
      description: >-
        После успешного списания средств с карты плательщика отправляется
        callback-запрос от платежного шлюза Multicard на URL партнера,
        отправленным в запросе на создание инвойса.


        Запрос отправляется со следующего IP: 195.158.26.90. При получении
        запроса система мерчанта должна проверить поле sign, либо установить
        ограничение на прием запросов только с этого IP-адреса.


        > sign формируется как md5-хеш от строки (без скобок):  

        {store_id}{invoice_id}{amount}{secret} 
          

        Для успешного ответа на указанный запрос необходимо ответить HTTP
        STATUS=200.


        Платеж отменяется (средства будут возвращены плательщику) в случае
        получения HTTP-статуса, отличного от 200 и/или если в теле ответа
        success != true. Значение из поля message будет отображено пользователю
        на странице неуспешной оплаты (чек).


        ``` json

        Пример неуспешного ответа (поле message будет отображено плательщику на
        странице оплаты):

        {

        "success": false,

        "message": "Не найден инвойс"

        }

         ```

        **При таймауте или получении HTTP-статуса 500, система Multicard
        заморозить транзакцию (средства будут заблокированы на карте
        плательщика). В таком случае, коллбэк-запрос может быть отправлен
        повторно. В связи с этим, при получении повторного запроса по инвойсу с
        тем же uuid (уникальный ID транзакции Multicard), если платеж успешный,
        необходимо вернуть успешный статус и обеспечить идемпотентность
        транзакции.**
      tags:
        - Оплата на платежной странице Multicard
      parameters: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                store_id:
                  type: integer
                  description: |+
                    ID кассы Партнера в системе Multicard

                amount:
                  type: integer
                  description: |+
                    Сумма платежа в тийинах

                invoice_id:
                  type: string
                  description: >-
                    Max length:255

                    Любой идентификатор заказа в системе Партнера. Будет
                    возвращен в callback-запросе. Также по нему можно искать
                    платежи в кабинете мерчанта
                billing_id:
                  type: string
                  description: Уникальный ID транзакции в системе Партнера
                payment_time:
                  type: string
                  description: |+
                    Время платежа в формате YYYY-mm-dd H:i:s

                phone:
                  type: string
                  description: |+
                    Телефон плательщика в формате 998XXXXXXXXX (при наличии)

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
                uuid:
                  type: string
                  description: Уникальный идентификатор транзакции в Multicard
                receipt_url:
                  type: string
                  description: Ссылка на фискальный чек
                sign:
                  type: string
                  description: |+
                    MD5 хеш от строки: {store_id}{invoice_id}{amount}{secret}

              required:
                - store_id
                - amount
                - invoice_id
                - billing_id
                - payment_time
                - phone
                - card_pan
                - ps
                - card_token
                - uuid
                - receipt_url
                - sign
              x-apidog-orders:
                - store_id
                - amount
                - invoice_id
                - billing_id
                - payment_time
                - phone
                - card_pan
                - ps
                - card_token
                - uuid
                - receipt_url
                - sign
            example:
              store_id: 6
              amount: 20000
              invoice_id: '2024864028760'
              billing_id: '20241214242009869794410864028760'
              payment_time: '2024-12-14 14:36:31'
              phone: '998930601725'
              card_pan: 860030******5959
              ps: uzcard
              card_token: 6225f3c93f7a880142782fa4
              uuid: e60d8ebc-b9fe-11ef-b159-005056b4367d
              receipt_url: >-
                https://dev-checkout.multicard.uz/check/e60d8ebc-b9fe-11ef-b159-005056b4367d
              sign: 553b4292b0f1d8e0e18e6daeb3af3761
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
      x-run-in-apidog: https://app.apidog.com/web/project/1022226/apis/api-19729300-run
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
