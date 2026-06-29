# Быстрая оплата (PaymeGo, ClickPass и др.)

## OpenAPI Specification

```yaml
openapi: 3.0.1
info:
  title: ''
  description: ''
  version: 1.0.0
paths:
  /payment/{uuid}/scanpay:
    put:
      summary: Быстрая оплата (PaymeGo, ClickPass и др.)
      deprecated: false
      description: >-
        Списания посредством QR-кода из приложения плательщика. Поддерживаются
        следующие приложения: Payme, Click, Uzum, Anorbank и Xazna.


        В случае успешного выполнения возвращается {PaymentModel}


        При получении таймаута, ошибки с кодом ERROR_DEBIT_UNKNOWN или
        http-статус 500 необходимо проверить статус платежа.
      tags:
        - Оплата на платежной странице Multicard
      parameters:
        - name: uuid
          in: path
          description: uuid инвойса для оплаты через PaymeGo, ClickPass и др.
          required: true
          example: 023874cd-b588-11f0-af7c-005056b4367d
          schema:
            type: string
        - name: X-Access-Token
          in: header
          description: ''
          required: false
          example: '{{multicard_token}}'
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                code:
                  type: string
                  description: Считанный код из приложения
              required:
                - code
              x-apidog-orders:
                - code
              x-apidog-ignore-properties: []
            example:
              code: '50512'
      responses:
        '200':
          description: ''
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                  data:
                    $ref: '#/components/schemas/paymentModel'
                x-apidog-orders:
                  - success
                  - data
                required:
                  - success
                  - data
                x-apidog-ignore-properties: []
          headers: {}
          x-apidog-name: Success
        '400':
          description: >-
            В случае показа неправильного QR для быстрого платежа, пользователь
            получает ошибку с кодом 400, и сообщение о неправильном QR 
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                    description: 'Статус запроса '
                  error:
                    type: object
                    properties:
                      code:
                        type: string
                        description: 'Кодовое сообщение '
                      details:
                        type: string
                        description: Сообщение об ошибке
                    required:
                      - code
                      - details
                    x-apidog-orders:
                      - code
                      - details
                    description: Ошибка
                    x-apidog-ignore-properties: []
                required:
                  - success
                  - error
                x-apidog-orders:
                  - success
                  - error
                x-apidog-ignore-properties: []
              example:
                success: false
                error:
                  code: ERROR_FIELDS
                  details: Неверный код, либо приложение не поддерживается
          headers: {}
          x-apidog-name: 'Показ неправильного QR '
      security: []
      x-apidog-folder: Оплата на платежной странице Multicard
      x-apidog-status: released
      x-run-in-apidog: https://app.apidog.com/web/project/1022226/apis/api-19729299-run
components:
  schemas:
    paymentModel:
      type: object
      properties:
        id:
          type: integer
          description: ID транзакции в Multicard
        uuid:
          type: string
          format: uuid
          description: UUID транзакции в Multicard
        store_id:
          type: integer
          description: ID кассы, выданный со стороны Multicard
        payment_amount:
          type: integer
        commission_type:
          type: string
          enum:
            - up
            - down
          x-apidog-enum:
            - value: up
              name: ''
              description: Комиссия сверху (снимается с плательщика)
            - value: down
              name: ''
              description: Комиссия снизу (удерживается с мерчанта внутри платежа)
          description: Тип комиссии
        commission_amount:
          type: integer
        total_amount:
          type: integer
          description: Сумма платежа в тийинах
        store_invoice_id:
          type: string
          description: >-
            Любой идентификатор заказа в системе Партнера. Будет возвращен в
            callback-запросе. Также по нему можно искать платежаи в кабинете
            мерчанта
        status:
          $ref: '#/components/schemas/PaymentStatusEnum'
          description: Статус транзакции
        callback_url:
          type: string
          description: URL для отправки callback-запроса
          nullable: true
        billing_id:
          type: string
          description: Уникальный ID транзакции в системе Партнера
          nullable: true
        phone:
          type: string
          description: Телефон плательщика в формате 998XXXXXXXXX (при наличии)
          nullable: true
        ps:
          description: Платежный сервис/система
          $ref: '#/components/schemas/PaymentServiceEnum'
        receipt_url:
          type: string
          description: Ссылка на платежный чек
          nullable: true
        kyc_data:
          type: object
          properties:
            last_name:
              type: string
              description: Фамилия плательщика
            first_name:
              type: string
              description: Имя плательщика
            middle_name:
              type: string
              description: Отчество плательщика
            passport:
              type: string
              description: Паспорт плательщика
            dob:
              type: string
              description: Дата рождения плательщика (YYYY-MM-DD)
            passport_expiry_date:
              type: string
              description: Дата истечения паспорта плательщика
          x-apidog-orders:
            - last_name
            - first_name
            - middle_name
            - passport
            - dob
            - passport_expiry_date
          required:
            - last_name
            - first_name
            - middle_name
            - passport
            - dob
            - passport_expiry_date
          x-apidog-ignore-properties: []
          nullable: true
        device_details:
          type: object
          properties:
            ip:
              type: string
              description: IP-адрес устройства клиента, с которого выполняется запрос
            user_agent:
              type: string
              description: >-
                Cтрока User-Agent, содержащая информацию о браузере,
                операционной системе и типе устройства клиента
          x-apidog-orders:
            - ip
            - user_agent
          description: Объект с информацией об устройстве клиента
          required:
            - ip
            - user_agent
          x-apidog-ignore-properties: []
          nullable: true
        details:
          type: object
          properties: {}
          x-apidog-orders: []
          description: >-
            Поля, необходимые для проведения платежа в биллинге. Используется
            только при оплате за услуги Paynet и МУНИС (мобильная связь,
            гос.платежи и т.п.). Список полей и их название зависит от
            конкретной услуги (store_id)
          x-apidog-ignore-properties: []
          nullable: true
        card_token:
          type: string
          nullable: true
        card_pan:
          type: string
          nullable: true
        split:
          anyOf:
            - $ref: '#/components/schemas/splitRequest'
            - type: 'null'
        multicard_user_id:
          type: integer
          description: >-
            ID пользователя в приложении Multicard (если оплата проведена через
            приложение)
          nullable: true
        ofd:
          anyOf:
            - $ref: '#/components/schemas/ofdRequest'
            - type: 'null'
        terminal_id:
          type: string
          description: ID терминала в платежной системе
          nullable: true
        merchant_id:
          type: string
          description: ID мерчанта в платежной системе
          nullable: true
        ps_uniq_id:
          type: string
          description: Reference number (RRN, RefNum) в платежной системе
          nullable: true
        ps_response_code:
          type: string
          description: Код ответа (ошибки) от платежной системы
          nullable: true
        ps_response_msg:
          type: string
          description: Описание ошибки от платежной системы
          nullable: true
        callback_message:
          type: string
          description: Текст ответа от биллинга мерчанта/поставщика
          nullable: true
        payment_time:
          type: string
          format: date-time
          description: >-
            Дата и время оплаты во временной зоне GMT+5 в формате YYYY-mm-dd
            H:s:s
          nullable: true
        refund_time:
          type: string
          format: date-time
          description: >-
            Дата и время возврата во временной зоне GMT+5 в формате YYYY-mm-dd
            H:s:s
        otp_hash:
          type: string
          description: >-
            Если значение не null, то требуется подтверждение транзакции через
            SMS-код (или 3DS)
          nullable: true
        clearing_id:
          type: integer
          nullable: true
        tax_receipt_id:
          type: integer
          nullable: true
        push_sent_at:
          type: string
          format: date-time
          nullable: true
        store:
          $ref: '#/components/schemas/storeModel'
        application:
          $ref: '#/components/schemas/applicationModel'
        tax:
          anyOf:
            - &ref_0
              $ref: '#/components/schemas/taxReceiptModel'
            - type: 'null'
        refund_tax:
          anyOf:
            - *ref_0
            - type: 'null'
        clearing:
          anyOf:
            - $ref: '#/components/schemas/clearingModel'
            - type: 'null'
        checkout_url:
          type: string
          description: URL страницы для оплаты
          nullable: true
        added_on:
          type: string
          format: date-time
      x-apidog-orders:
        - id
        - uuid
        - store_id
        - payment_amount
        - commission_type
        - commission_amount
        - total_amount
        - store_invoice_id
        - status
        - callback_url
        - billing_id
        - phone
        - ps
        - receipt_url
        - kyc_data
        - device_details
        - details
        - card_token
        - card_pan
        - split
        - multicard_user_id
        - ofd
        - terminal_id
        - merchant_id
        - ps_uniq_id
        - ps_response_code
        - ps_response_msg
        - callback_message
        - payment_time
        - refund_time
        - otp_hash
        - clearing_id
        - tax_receipt_id
        - push_sent_at
        - store
        - application
        - tax
        - refund_tax
        - clearing
        - checkout_url
        - added_on
      required:
        - id
        - uuid
        - store_id
        - payment_amount
        - commission_type
        - commission_amount
        - total_amount
        - store_invoice_id
        - status
        - callback_url
        - billing_id
        - phone
        - ps
        - receipt_url
        - kyc_data
        - device_details
        - details
        - card_token
        - card_pan
        - split
        - multicard_user_id
        - ofd
        - terminal_id
        - merchant_id
        - ps_uniq_id
        - ps_response_code
        - ps_response_msg
        - callback_message
        - payment_time
        - refund_time
        - otp_hash
        - clearing_id
        - tax_receipt_id
        - push_sent_at
        - store
        - application
        - tax
        - refund_tax
        - clearing
        - checkout_url
        - added_on
      x-apidog-ignore-properties: []
      x-apidog-folder: ''
    clearingModel:
      type: object
      properties:
        id:
          type: integer
        merchant: &ref_2
          $ref: '#/components/schemas/merchantModel'
          description: Информация о клиенте (мерчанте)
        recipient_info: &ref_1
          $ref: '#/components/schemas/merchantAccount'
        sender_info: *ref_1
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
        - amount
        - details
        - status
        - payment_time
        - added_on
        - updated_on
        - receipt_url
      x-apidog-ignore-properties: []
      x-apidog-folder: ''
    merchantAccount:
      type: object
      properties:
        id:
          type: integer
          description: ID банковских реквизитов
        uuid:
          type: string
          format: uuid
          description: UUID банковских реквизитов
        tin:
          type: string
          description: ИНН или ПИНФЛ
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
        - official_name
        - mfo
        - account_no
        - address
        - director
        - director_pinfl
        - vat_payer
        - is_commitent
        - active
      x-apidog-ignore-properties: []
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
        - contract_id
        - bank_account
      x-apidog-ignore-properties: []
      x-apidog-folder: ''
    taxReceiptModel:
      type: object
      properties:
        receipt:
          type: object
          properties: {}
          x-apidog-orders: []
          description: Объект чека, отправленный в ГНК
          x-apidog-ignore-properties: []
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
        - qr_url
        - is_refund
      x-apidog-ignore-properties: []
      x-apidog-folder: ''
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
        - wallet_sender_account
        - wallet_overdraft
        - wallet_contract_num
        - otp_required
        - otp_gateway
        - sms_nickname
      x-apidog-ignore-properties: []
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
        merchant: *ref_2
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
              x-apidog-ignore-properties: []
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
            - num
            - date
            - service
            - fee
            - edm_document_id
            - edm_status
          x-apidog-ignore-properties: []
          nullable: true
        merchant_account: *ref_1
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
      x-apidog-ignore-properties: []
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
          - name
          - value
          - key
          - suggested
        x-apidog-ignore-properties: []
      x-apidog-folder: ''
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
          - total
          - package_code
          - name
        x-apidog-ignore-properties: []
      x-apidog-folder: ''
    splitRequest:
      type: array
      items:
        type: object
        properties:
          type:
            type: string
            enum:
              - account
              - wallet
              - card
            x-apidog-enum:
              - value: account
                name: ''
                description: Расчетный счет
              - value: wallet
                name: ''
                description: Депозит приложения
              - value: card
                name: ''
                description: Банковская карта
            description: Тип получателя
          amount:
            type: integer
            description: Сумма платежа
          details:
            type: string
            description: Детали платежа
          recipient:
            type: string
            description: >-
              Получатель. Если type=account, то передается uuid банковских
              реквизитов. По-умолчанию подставляются банковские реквизиты
              мерчанта.
        x-apidog-orders:
          - type
          - amount
          - details
          - recipient
        required:
          - type
          - amount
          - details
        x-apidog-ignore-properties: []
      x-apidog-folder: ''
    PaymentServiceEnum:
      type: string
      enum:
        - uzcard
        - humo
        - visa
        - mastercard
        - account
        - payme
        - click
        - uzum
        - anorbank
        - oson
        - alif
        - xazna
        - beepul
        - trastpay
        - sbp
      x-apidog-enum:
        - value: uzcard
          name: ''
          description: ''
        - value: humo
          name: ''
          description: ''
        - value: visa
          name: ''
          description: ''
        - value: mastercard
          name: ''
          description: ''
        - value: account
          name: ''
          description: ''
        - value: payme
          name: ''
          description: ''
        - value: click
          name: ''
          description: ''
        - value: uzum
          name: ''
          description: ''
        - value: anorbank
          name: ''
          description: ''
        - value: oson
          name: ''
          description: ''
        - value: alif
          name: ''
          description: ''
        - value: xazna
          name: ''
          description: ''
        - value: beepul
          name: ''
          description: ''
        - value: trastpay
          name: ''
          description: ''
        - value: sbp
          name: ''
          description: ''
      description: Платежная система или сервис
      x-apidog-folder: ''
    PaymentStatusEnum:
      type: string
      enum:
        - draft
        - progress
        - billing
        - success
        - error
        - revert
      x-apidog-enum:
        - value: draft
          name: ''
          description: транзакция не подтверждена
        - value: progress
          name: ''
          description: транзакция в процессе списания в платежной системе
        - value: billing
          name: ''
          description: идет отправка запроса в биллинг поставщика
        - value: success
          name: ''
          description: успешно
        - value: error
          name: ''
          description: >-
            ошибка при списании или отправки запроса в биллинг поставщика.
            транзакция не успешна
        - value: revert
          name: ''
          description: возврат
      x-apidog-folder: ''
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
servers: []
security:
  - bearer: []

```
