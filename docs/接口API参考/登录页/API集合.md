##
```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/corp/common/get_public_configuration

<!-- 响应参数 -->
{
  "version": "11.7.2",
  "calling_code_list": [
    {
      "label": "",
      "children": [
        {
          "text": {
            "zh_cn": "中国 +86",
            "en_us": "China +86",
            "zh_tw": "中國 +86"
          },
          "value": "+86"
        },
        {
          "text": {
            "zh_cn": "中国台湾 +886",
            "en_us": "Taiwan +886",
            "zh_tw": "中國台灣 +886"
          },
          "value": "+886"
        },
        {
          "text": {
            "zh_cn": "中国香港 +852",
            "en_us": "Hong Kong +852",
            "zh_tw": "中國香港 +852"
          },
          "value": "+852"
        },
        {
          "text": {
            "zh_cn": "中国澳门 +853",
            "en_us": "Macao +853",
            "zh_tw": "中國澳門 +853"
          },
          "value": "+853"
        }
      ]
    }
  ],
  "qixin_search_service": true,
  "captcha": {
    "type": "svg",
    "params": {}
  },
  "pki": {
    "algorithm": "rsa",
    "keys": {
      "public_key": "-----BEGIN PUBLIC KEY-----\nMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAMkqCoxgdIfhTffG7jBB0UtA9BjjpT/i\nSkLIku8C1PN+DB3e3ngiOaTUCHB9uyrtyr6NnJwtY/N/8NJSsT07BYECAwEAAQ==\n-----END PUBLIC KEY-----\n"
    }
  },
  "order_info": true,
  "product_center": true,
  "tenant_register": true,
  "corp_coop": true,
  "customize_better_version": "https://www.jiandaoyun.com/f/5a1fd19603f8b9264c1f96df",
  "howxm_project_id": "2f2ef255-4629-4d86-9da4-5ed9e35be64d",
  "member_auto_sync": true,
  "support": true,
  "contact_phone": true,
  "platform_sms": true,
  "coin": true,
  "register_persona": true,
  "help_docs": {
    "coin": "https://hc.jiandaoyun.com/doc/12598"
  },
  "links": {
    "purchase_coin": ""
  }
}
```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/authn/otp/generate_captcha?_=1787027864635

<!-- 响应参数 -->
{
  "data": "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"188\" height=\"60\" viewBox=\"0,0,188,60\"><rect width=\"100%\" height=\"100%\" fill=\"#cc9966\"/><path d=\"M17 42 C80 6,90 27,187 57\" stroke=\"#e9e995\" fill=\"none\"/><path fill=\"#544b58\" d=\"M68.92 45.45L68.86 45.40L68.76 45.30Q64.68 45.52 63.69 43.20L63.85 43.36L63.84 43.35Q64.58 42.37 66.14 40.59L66.17 40.62L66.03 40.48Q66.50 42.47 69.51 42.58L69.54 42.62L69.68 42.76Q72.98 42.74 74.61 41.41L74.61 41.40L74.68 41.47Q76.02 39.80 75.94 37.10L75.86 37.02L75.98 37.14Q75.90 32.15 71.10 32.38L70.92 32.19L70.99 32.26Q68.26 32.50 66.51 33.75L66.49 33.74L66.10 33.51L66.00 33.47L65.82 33.30Q66.32 30.29 66.20 27.55L66.07 27.42L66.05 27.40Q66.02 24.86 65.68 21.54L65.78 21.64L65.74 21.60Q69.32 22.49 73.13 22.37L73.18 22.42L73.19 22.43Q76.94 22.30 80.52 21.05L80.49 21.02L79.91 22.53L80.01 22.63Q79.59 23.27 79.40 24.11L79.52 24.23L79.47 24.18Q76.29 25.15 72.40 25.15L72.45 25.20L72.40 25.14Q71.10 25.30 69.62 25.14L69.53 25.05L69.52 25.04Q69.36 25.65 69.02 30.10L69.04 30.13L69.03 30.11Q69.97 29.76 72.03 29.61L72.09 29.67L72.05 29.63Q75.66 29.73 77.10 31.41L77.25 31.55L77.25 31.56Q78.64 33.10 78.91 36.95L78.82 36.86L78.79 36.82Q79.10 41.06 77.88 43.07L77.95 43.14L77.93 43.13Q75.83 44.94 72.51 45.17L72.54 45.19L72.46 45.11Q71.42 45.14 68.76 45.29ZM74.91 47.68L74.80 47.57L74.85 47.62Q78.32 47.74 80.22 46.63L80.30 46.71L80.30 46.71Q81.17 45.07 81.17 42.98L81.20 43.00L81.26 43.07Q81.27 40.00 80.32 35.88L80.22 35.79L80.23 35.80Q79.79 33.98 78.57 32.57L78.72 32.72L78.73 32.77L78.40 32.36L78.05 32.16L78.07 32.19Q77.87 31.57 77.23 30.89L77.26 30.92L77.16 30.86L77.22 30.92Q75.73 29.28 72.04 29.28L71.89 29.13L71.69 29.20L71.72 29.22Q71.72 28.58 71.91 27.47L71.91 27.47L72.03 27.59Q76.98 27.67 81.02 26.03L80.92 25.94L80.89 25.91Q81.50 24.69 82.45 21.88L82.32 21.74L80.25 22.72L80.19 22.66Q80.70 21.64 81.19 20.50L81.16 20.47L81.19 20.49Q77.28 21.87 73.16 21.95L73.19 21.98L73.12 21.91Q69.10 22.11 65.14 21.01L65.21 21.07L65.20 21.07Q65.86 25.46 65.86 29.88L65.88 29.89L65.86 29.87Q65.77 31.69 65.66 33.59L65.60 33.54L65.68 33.62Q65.97 33.83 66.54 34.14L66.59 34.19L66.61 34.20Q66.91 33.93 67.56 33.59L67.42 33.45L67.47 33.50Q67.48 34.31 67.25 35.83L67.33 35.91L67.28 35.86Q67.71 36.06 68.12 36.29L68.06 36.22L68.00 36.16Q70.89 34.52 72.75 34.52L72.63 34.40L72.80 34.57Q74.11 34.54 75.29 35.19L75.15 35.06L75.18 35.08Q75.56 36.15 75.60 37.10L75.64 37.14L75.58 37.09Q75.65 39.66 74.62 40.77L74.63 40.77L74.58 40.73Q73.28 42.01 71.03 42.24L71.08 42.29L70.96 42.17Q70.14 42.37 69.45 42.30L69.46 42.30L69.42 42.27Q68.56 42.13 67.80 41.79L67.84 41.83L67.68 41.48L67.68 41.70L67.67 41.69Q66.92 41.25 66.38 39.91L66.28 39.81L66.35 39.88Q64.94 41.21 63.26 43.34L63.38 43.46L63.27 43.35Q63.69 44.11 64.53 45.06L64.39 44.92L64.40 44.94Q65.45 46.75 68.27 47.25L68.42 47.40L68.26 47.24Q69.57 47.59 74.94 47.71Z\"/><path fill=\"#263526\" d=\"M106.61 31.82L106.59 31.81L106.70 31.91Q108.43 32.01 110.26 32.01L110.32 32.07L110.31 32.06Q112.05 32.01 113.88 31.86L113.82 31.81L113.85 31.83Q113.82 32.57 113.82 33.29L113.77 33.24L113.64 34.44L113.64 34.43Q111.40 34.56 106.64 34.68L106.56 34.59L106.52 34.55Q106.71 40.68 105.57 45.67L105.41 45.51L105.54 45.64Q103.34 46.34 101.86 47.13L101.79 47.07L101.73 47.00Q104.01 40.15 103.74 33.03L103.75 33.04L103.89 33.17Q103.59 25.98 100.92 19.32L100.86 19.26L100.86 19.26Q104.58 21.84 109.34 22.07L109.26 21.99L109.45 22.18Q113.92 22.15 118.07 20.40L118.16 20.49L118.23 20.56Q117.88 21.32 117.65 22.11L117.71 22.18L117.24 23.80L117.20 23.76Q114.40 24.76 111.43 24.92L111.57 25.05L111.45 24.94Q108.56 25.13 105.70 24.41L105.73 24.43L105.69 24.39Q106.42 27.83 106.61 31.82ZM118.78 19.79L118.86 19.87L118.87 19.87Q114.19 21.94 109.32 21.63L109.27 21.58L109.22 21.53Q103.97 21.30 100.24 18.45L100.17 18.38L100.29 18.50Q103.09 25.37 103.39 32.98L103.55 33.14L103.45 33.04Q103.71 40.92 101.20 47.69L101.36 47.86L101.25 47.74Q101.92 47.35 103.29 46.71L103.32 46.73L103.23 46.64Q103.01 47.37 102.44 48.90L102.54 49.00L102.57 49.03Q104.90 47.96 107.79 47.47L107.74 47.43L107.75 47.43Q108.28 42.47 108.47 36.76L108.51 36.80L108.48 36.78Q110.38 36.85 112.21 36.85L112.23 36.87L112.09 36.73Q114.02 36.87 115.84 37.06L115.76 36.98L115.83 37.05Q115.59 35.97 115.59 35.06L115.60 35.06L115.59 33.15L115.65 33.21Q114.94 33.37 114.10 33.37L114.17 33.44L114.03 33.31Q114.15 32.55 114.22 31.52L114.17 31.46L114.16 31.46Q112.78 31.64 111.37 31.64L111.43 31.70L111.46 31.72Q110.06 31.74 108.69 31.70L108.60 31.60L108.52 29.39L108.52 29.40Q108.51 28.32 108.40 27.25L108.30 27.16L108.34 27.20Q109.30 27.24 110.29 27.24L110.33 27.29L110.44 27.39Q115.13 27.33 118.75 25.43L118.73 25.40L118.70 25.37Q119.14 23.49 120.05 20.86L120.19 21.00L120.04 20.85Q118.75 21.70 117.95 22.04L118.07 22.16L117.97 22.06Q118.20 21.26 118.70 19.70Z\"/><path d=\"M17 25 C76 4,110 45,182 20\" stroke=\"#77e353\" fill=\"none\"/><path d=\"M8 51 C90 16,95 27,178 19\" stroke=\"#8ca3e7\" fill=\"none\"/><path fill=\"#4a4743\" d=\"M145.59 34.69L145.56 34.66L145.55 34.65Q141.53 34.62 140.96 37.67L140.84 37.55L140.78 37.49Q140.76 38.73 140.95 39.61L140.81 39.46L140.88 39.53Q140.86 40.28 141.32 41.69L141.31 41.67L141.31 41.68Q142.35 44.81 145.66 44.66L145.69 44.69L145.67 44.67Q147.72 44.77 149.05 43.21L149.18 43.34L149.16 43.33Q150.31 41.77 150.31 39.71L150.21 39.62L150.39 39.79Q150.37 39.05 150.22 37.98L150.31 38.07L150.24 38.00Q150.24 37.17 149.82 36.48L149.71 36.38L149.90 36.57Q148.23 34.85 145.64 34.74ZM150.47 53.19L150.56 53.28L150.40 53.13Q148.55 53.68 140.48 54.06L140.46 54.04L140.60 54.18Q138.94 54.27 137.30 53.43L137.33 53.46L137.21 53.34Q138.15 52.49 139.98 50.70L139.90 50.62L140.00 50.72Q142.40 51.75 144.49 51.56L144.44 51.51L144.32 51.39Q147.39 51.30 148.22 51.03L148.13 50.94L148.20 51.00Q150.29 50.16 150.29 48.37L150.38 48.47L150.44 48.53Q150.29 48.15 150.22 47.92L150.30 48.00L150.24 46.45L150.27 46.49Q150.22 45.68 150.22 44.88L150.09 44.75L150.26 44.92Q148.79 47.06 145.40 47.06L145.41 47.07L145.42 47.08Q141.52 47.07 139.88 44.86L139.83 44.80L139.89 44.87Q138.77 43.32 137.97 38.91L138.01 38.95L138.04 38.98Q137.65 37.37 137.65 35.96L137.71 36.03L137.76 36.07Q137.75 34.28 138.67 33.29L138.62 33.24L138.62 33.24Q140.27 31.81 144.92 31.81L145.04 31.93L146.69 32.02L146.67 32.00Q149.90 32.38 151.23 34.32L151.14 34.22L151.18 34.26Q151.34 33.40 151.76 31.76L151.79 31.79L151.73 31.73Q153.75 31.54 155.50 30.81L155.34 30.66L155.49 30.81Q152.67 36.75 152.67 43.98L152.65 43.96L152.83 44.14Q152.78 46.75 153.16 49.38L153.03 49.25L153.04 49.26Q153.23 49.95 153.16 50.63L153.18 50.66L153.18 50.65Q152.96 52.00 151.74 52.76L151.78 52.80L151.89 52.90Q151.27 53.05 150.43 53.16ZM153.50 56.19L153.48 56.17L153.40 56.09Q155.02 56.30 155.78 55.35L155.74 55.31L155.67 55.23Q156.02 54.30 155.91 53.57L155.87 53.53L155.86 53.52Q155.73 52.86 155.54 52.10L155.61 52.17L155.71 52.27Q154.41 46.97 154.75 41.72L154.68 41.64L154.71 41.68Q155.19 36.38 157.33 31.54L157.15 31.37L155.21 32.47L155.20 32.46Q155.29 31.71 155.52 31.14L155.55 31.17L156.16 30.14L156.07 30.06Q153.82 31.12 151.61 31.54L151.52 31.44L151.55 31.47Q151.17 32.27 151.01 33.30L150.97 33.25L150.98 33.26Q148.86 31.34 144.83 31.34L144.88 31.39L143.15 31.41L143.19 31.45Q139.90 31.40 138.22 32.81L138.24 32.82L138.31 32.89Q137.25 33.74 137.29 35.64L137.35 35.70L137.39 35.74Q137.53 38.31 138.55 42.58L138.54 42.57L138.45 42.47Q138.95 44.42 140.17 45.82L140.19 45.85L140.42 46.08L140.41 46.07L140.50 46.16Q141.55 48.39 144.14 48.93L144.28 49.06L144.29 49.07Q145.69 49.37 147.14 49.41L147.04 49.32L147.04 49.31Q148.78 49.34 149.85 48.96L149.81 48.92L149.98 49.09Q149.22 50.50 146.55 50.84L146.60 50.89L146.67 50.96Q145.55 51.06 144.87 51.06L144.87 51.06L144.39 50.97L144.46 51.03Q141.72 51.11 140.05 50.16L139.96 50.08L138.35 51.89L138.24 51.78Q137.47 52.69 136.63 53.64L136.66 53.66L136.62 53.62Q137.60 54.15 138.59 54.34L138.59 54.34L137.78 55.28L137.62 55.12Q140.44 56.30 146.08 56.30L146.14 56.37L146.23 56.23L146.27 56.26Q149.83 56.21 153.41 56.10ZM147.43 36.95L147.45 36.97L147.57 37.09Q148.86 37.01 149.78 37.43L149.82 37.47L149.66 37.31Q149.99 37.95 150.06 38.63L150.02 38.59L150.10 38.66Q149.97 38.80 149.93 39.72L150.02 39.80L150.07 39.85Q150.00 41.77 148.90 43.06L148.87 43.03L148.82 42.98Q147.73 44.48 145.83 44.44L145.76 44.38L145.69 44.31Q144.19 44.18 143.35 43.76L143.37 43.78L143.52 43.93Q143.08 42.73 143.04 41.70L142.97 41.62L142.96 41.61Q142.90 37.33 147.51 37.03Z\"/><path fill=\"#2f3634\" d=\"M33.30 45.00L33.22 44.92L33.23 44.93Q29.01 44.33 27.19 42.89L27.16 42.85L27.24 42.94Q25.37 41.45 24.95 38.44L25.05 38.54L25.02 38.51Q24.98 38.13 24.64 33.37L24.56 33.29L24.65 33.38Q24.70 32.21 24.62 30.95L24.51 30.84L24.55 30.88Q24.36 25.82 26.57 24.07L26.53 24.03L26.65 24.15Q29.19 21.94 36.96 21.25L37.11 21.40L37.02 21.31Q38.40 21.21 40.04 21.25L40.01 21.22L40.01 21.21Q39.96 21.17 42.93 21.17L42.92 21.15L42.99 21.23Q43.77 21.17 45.49 21.33L45.61 21.45L45.54 21.38Q44.99 22.28 43.96 25.36L43.94 25.34L43.95 25.35Q42.00 24.27 38.99 24.27L38.95 24.23L38.92 24.20Q38.17 24.21 37.37 24.29L37.30 24.22L37.37 24.28Q32.17 24.65 29.66 26.63L29.71 26.68L29.79 26.76Q27.64 28.22 27.49 31.57L27.49 31.58L27.50 31.59Q27.56 32.29 27.60 33.55L27.50 33.46L27.52 33.47Q27.70 38.37 30.09 40.46L30.12 40.49L29.97 40.34Q32.26 42.40 37.24 42.70L37.31 42.77L37.28 42.74Q40.39 42.88 43.40 41.09L43.42 41.11L43.30 40.99Q43.97 43.79 44.57 45.16L44.60 45.19L44.56 45.15Q42.43 45.38 40.79 45.34L40.81 45.36L40.88 45.43Q36.44 45.40 33.32 45.02ZM48.39 48.71L48.36 48.68L48.29 48.61Q46.47 45.57 45.78 42.98L45.86 43.05L45.78 42.98Q45.46 43.34 44.47 43.72L44.37 43.62L44.43 43.68Q44.06 42.85 43.87 42.02L43.99 42.14L43.52 40.34L43.63 40.45Q40.35 42.50 37.23 42.39L37.23 42.39L37.18 42.33Q32.90 42.13 30.58 40.38L30.58 40.38L30.66 40.45Q29.44 38.56 29.52 35.62L29.55 35.66L29.42 35.52Q29.66 31.12 32.13 28.98L32.09 28.94L32.11 28.97Q34.16 26.98 38.85 26.37L38.88 26.41L38.97 26.50Q39.66 26.27 40.38 26.27L40.36 26.26L40.41 26.30Q43.15 26.26 45.24 27.78L45.26 27.80L45.41 27.95Q45.78 25.69 47.00 22.46L46.99 22.45L47.11 22.57Q46.86 22.58 46.31 22.53L46.20 22.42L46.24 22.47Q45.76 22.48 45.49 22.48L45.45 22.43L45.36 22.34Q45.60 21.86 46.06 20.87L46.08 20.89L46.01 20.82Q45.73 20.77 43.24 20.70L43.37 20.83L43.36 20.82Q40.75 20.63 40.10 20.66L40.22 20.78L40.23 20.79Q29.60 21.01 26.14 23.60L26.29 23.76L26.18 23.64Q24.11 25.42 24.11 29.34L24.25 29.48L24.22 29.45Q24.26 30.59 24.37 33.29L24.29 33.21L24.28 33.20Q24.43 36.78 24.66 38.50L24.70 38.53L24.71 38.55Q24.97 41.54 26.57 43.07L26.55 43.05L26.59 43.09Q27.05 43.74 28.30 44.88L28.40 44.97L28.44 45.02Q31.75 46.84 37.00 47.60L36.97 47.57L37.00 47.60Q43.31 48.43 48.30 48.62Z\"/></svg>"
}
```


```
https://www.jiandaoyun.com/portal/authn/otp/send_sms

<!-- 请求参数 -->
{
  "phone": "+86-15355381414",
  "captcha": "HAAA/fYO/+TRgBXI8XIKpZIuc482Lz5gubxU+5XKRgs="
}

<!-- 响应参数 -->
无
```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/signin

<!-- 请求参数 -->
{
  "username": "+86-15355381414",
  "phone_code": "425290",
  "is_remember": true,
  "suite_id": 1
}
<!-- 响应参数 -->
{
  "user": {
    "_id": "6a7f3132e6a0aba27cdb3d2b",
    "username": "sys_6a7f3132e6a0aba27cdb3d2a",
    "nickname": "李同学",
    "is_email_verified": false,
    "phone": "15355381414",
    "is_phone_verified": true
  },
  "is_register": false,
  "invitations": [],
  "suite": {
    "_id": 1,
    "type": 1,
    "name": "简道云",
    "pc_uri": "https://www.jiandaoyun.com/dashboard",
    "mobile_uri": "https://www.jiandaoyun.com/dashboard",
    "management_uri": "https://www.jiandaoyun.com/profile",
    "icon_uri": "https://g.jdycdn.com/shared/images/logo/jdy_logo.png"
  }
}
```

```
<!-- 接口 -->
https://www.jiandaoyun.com/corp/v2/login_user_info

<!-- 请求参数 -->
{
  "suite_id": 1
}
<!-- 响应参数 -->
{
  "tenant": {
    "id": "6a7f3132e6a0aba27cdb3d2b",
    "corp_id": "6a7f3132e6a0aba27cdb3d2b",
    "integrate_type": "Default",
    "has_link": false,
    "has_name_modify": false,
    "has_crm": false,
    "has_kms": true,
    "name": "重庆万柯互联网科技有限责任公司",
    "owner_id": "6a7f3132e6a0aba27cdb3d2b",
    "modules": {
      "aggregate_batch_migration_status": 1,
      "aggregate_migration_status": 1,
      "ai_lab": true,
      "member_api_key": true,
      "member_api_key_license": true,
      "trigger_charge_migration_status": 1,
      "trigger_migration_status": 2,
      "kms": true,
      "app_locale": false,
      "ai_lab_beta": true
    },
    "wx_enable": false,
    "wxwork": {
      "open_data": false
    },
    "size": 1,
    "mfa_enabled": false,
    "is_wxwork_suite_corp": false,
    "watermark": {
      "enabled": false,
      "color": "light",
      "density": "normal"
    },
    "op": {
      "crm_videos_1": true,
      "crm_videos_2": true
    },
    "corp_theme": {
      "app_navi_color": "dark"
    },
    "timezone": {
      "label": "Asia/Shanghai",
      "etc": "Etc/GMT-8"
    },
    "region": {
      "use12Hour": false,
      "dateTimeFormat": {
        "ymd_hms": "YYYY-MM-DD HH:mm:ss",
        "ymd_hm": "YYYY-MM-DD HH:mm",
        "ymd": "YYYY-MM-DD",
        "ym": "YYYY-MM",
        "hms": "HH:mm:ss",
        "hm": "HH:mm"
      },
      "weekStart": 1,
      "yearStart": 4,
      "decimalSeparator": ".",
      "thousandSeparator": ","
    },
    "entity_localization": false,
    "ai_capability": {
      "ai_lab": {
        "auto_fill": true,
        "knowledge_qa": true,
        "speech_recognition": true,
        "rich_text_assist": true,
        "field_auto_fill": true,
        "field_suggestion": true,
        "aggregation_formula": true,
        "automation_auto_match": true,
        "llm_node": true,
        "llm_multi_modal_node": true,
        "automation_ai_agent": true,
        "generate_modify_function": true
      }
    }
  },
  "vip": {
    "upgradable_addons": [],
    "vip_buffer_state": "normal",
    "pack": {
      "display_level": "trial",
      "display_level_i18n": {
        "zh_cn": "试用版",
        "zh_tw": "試用版",
        "en_us": "Trial",
        "de_de": "Versuch",
        "es_es": "Ensayo",
        "fr_fr": "Procès",
        "id_id": "Uji coba",
        "ja_jp": "トライアル",
        "km_kh": "ការសាកល្បង",
        "ko_kr": "재판",
        "pt_pt": "Julgamento",
        "ru_ru": "Пробный",
        "th_th": "การทดลอง",
        "vi_vn": "Thử nghiệm"
      },
      "level": "trial",
      "service": "basic",
      "users": 30,
      "managers": -1,
      "sys_managers": 1,
      "app": -1,
      "form_data": 300000,
      "data": 750000,
      "excel_import": 5242880,
      "data_xlsx_import": 20971520,
      "data_xls_import": 5242880,
      "file_upload": 128849018880,
      "file_storage": -1,
      "file_max_size": 20971520,
      "file_zip": 1,
      "import_file": 1,
      "data_api": 1,
      "aggregate": 20,
      "form_aggregate": 120,
      "data_backup": 180,
      "logo": 0,
      "user_sync": 1,
      "app_bridge": 1,
      "app_manage_group": 1,
      "permission_management": 1,
      "submit_prompt": 1,
      "print": -1,
      "signature": 1,
      "aggregation": 100,
      "form_aggregation": 1,
      "etl": 50,
      "trigger": 120,
      "trigger_times": 500000,
      "automation_execute_times": 1000000,
      "custom_theme": 1,
      "sso": 1,
      "custom_login": 1,
      "threshold": 1,
      "public_link": 0,
      "dash_style": 1,
      "bpa": 20,
      "open": 1,
      "open_private_plugin": 1,
      "pay": 1,
      "corp_coop": 60,
      "corp_coop_docker": 500,
      "member_freeze": 1,
      "corp_theme": 1,
      "corp_metrics": 1,
      "chat_bot": 1,
      "dash_advanced_charts": 1,
      "watermark": 1,
      "attachment_control": 0,
      "app_attachment_control": 1,
      "data_mask": 1,
      "kms": 1,
      "workbench_advanced": 1,
      "crm": 0,
      "corp_executive": 1,
      "analytics": 1,
      "form_hierarchy_view": 1,
      "analytics_editor": 0,
      "analytics_viewer": 0,
      "automation_loop_node": 1,
      "template_center": 1,
      "official_template": 1,
      "plugin_store": 1,
      "official_plugin": 1,
      "official_data_source": 1,
      "open_application": 1,
      "workflow_simulation": 1,
      "automation_http_node": 1,
      "form_data_page": 1,
      "automation_advanced_node": 1,
      "form_openness_enhancement": 1,
      "ai_compute": 500
    },
    "state": 0,
    "trial_date": "2026-08-28T16:00:00.000Z",
    "expire_date": 0,
    "has_trial": false,
    "has_purchase": true,
    "has_upload_certificate": true,
    "is_buffer": false,
    "is_buffer_expired": false,
    "is_addon_expired": false,
    "is_vip_expired": false,
    "level_expired_days": 10,
    "vip_buf_days_balance": 15
  },
  "platformPlan": {
    "plan": "trial",
    "plan_users": -1,
    "plan_addon": {},
    "plan_gift": {},
    "plan_trial": "2026-08-28T16:00:00.000Z",
    "is_paid": false,
    "is_trial": true,
    "is_free": false,
    "pack": {
      "corp_coop": 60,
      "corp_coop_docker": 500,
      "custom_login": 1,
      "data_api": 1,
      "log_api": 0,
      "sso": 1,
      "corp_executive": 1,
      "user_sync": 1,
      "member_freeze": 1,
      "open": 1,
      "plugin_store": 1,
      "data_source_store": 1,
      "open_private_plugin": 1,
      "open_application": 1,
      "member_api": 1,
      "sys_admin": 5,
      "watermark": 1,
      "corp_theme": 1,
      "ai_lab": 1,
      "excel_import": 5242880
    },
    "upgradable_addons": []
  },
  "member": {
    "member_id": "6a7f3132e6a0aba27cdb3d2b",
    "username": "sys_6a7f3132e6a0aba27cdb3d2a",
    "nickname": "李同学",
    "is_admin": true,
    "is_owner": true,
    "has_app_edit_permission": true,
    "is_app_builder_manager": true,
    "is_open_platform_manager": true,
    "dept": [
      {
        "_id": "6a7f31d6f33abed79701e2dc",
        "dept_no": 1,
        "parent_no": 0,
        "name": "重庆万柯互联网科技有限责任公司",
        "path": [],
        "type": 0,
        "status": 1,
        "seq": 0,
        "departmentId": 1,
        "parentId": 0,
        "order": 0
      }
    ],
    "has_help": true,
    "has_chat_bot": true,
    "has_template_center": true,
    "has_file_manage": false,
    "type": 0,
    "visible_suite_ids": [
      1
    ]
  },
  "user": {
    "_id": "6a7f3132e6a0aba27cdb3d2b",
    "username": "sys_6a7f3132e6a0aba27cdb3d2a",
    "nickname": "李同学",
    "phone": "15355381414",
    "is_phone_verified": true,
    "email": "",
    "is_email_verified": false,
    "has_password": false,
    "guide": {
      "dashboard": true,
      "form_edit": true,
      "flow_enable": true,
      "report_edit": true,
      "etl_edit": true,
      "flow_trial": true,
      "crm_videos": true,
      "mobile_x_video": true,
      "mobile_x_jdy": true,
      "mobile_x_app": true,
      "mobile_x_auth": true,
      "dash_edit": true,
      "etl_guide_done": true,
      "persona": true,
      "automation_design": true,
      "bpa_with_data": true,
      "corp_coop_mobile": true,
      "crm": true
    },
    "reg_time": "2026-08-14T15:16:02.798Z",
    "login_type": "user",
    "locale": "zh_cn",
    "is_sync": false,
    "owned_tenant_list": [
      {
        "id": "6a7f3132e6a0aba27cdb3d2b",
        "name": "重庆万柯互联网科技有限责任公司"
      }
    ],
    "jsy_migration_type": 0
  },
  "auth_suite_id": 1,
  "suite_modules": [
    {
      "name": "简道云",
      "key": "dashboard",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "managementUrl": "https://www.jiandaoyun.com/profile",
      "icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_logo.png",
      "suiteId": 1,
      "enable": true,
      "plan_level": "1/trial",
      "plan_name": "试用版",
      "is_beta": false,
      "description": "支持低成本快速搭建企业级管理应用，通过功能组合，灵活实现从数据采集、流转到处理、分析的全场景覆盖。",
      "help_doc": "https://hc.jiandaoyun.com/doc/12534",
      "vip_url": "/portal/tenant/6a7f3132e6a0aba27cdb3d2b/admin?_from=1#/vip",
      "guide_img": "https://g.jdycdn.com/shared/images/guide/jdy_guide.png",
      "guide_description": "简道云支持低成本快速搭建企业级管理应用，通过功能组合，灵活实现从数据采集、流转到处理、分析的全场景覆盖。"
    }
  ],
  "navigation_modules": [
    {
      "key": "dashboard",
      "suiteId": 1,
      "name": "工作台",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_nav_icon.png",
      "pc_icon_bg": "linear-gradient(146deg,rgba(59,173,74,.12) 14.15%,rgba(59,173,74,.1) 50.08%,rgba(59,173,74,.12) 93.63%)",
      "mobile_visible": true,
      "mobile_icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_nav_icon_mobile.png",
      "is_beta": false
    },
    {
      "key": "kms",
      "name": "知识库",
      "url": "https://kms.jiandaoyun.com",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/kms_nav_icon.png",
      "pc_icon_bg": "radial-gradient(111.23% 111.23% at 8.33% 7.08%,rgba(250,133,30,.18) 0,rgba(250,133,30,.1) 50%,rgba(250,133,30,.16) 100%)",
      "mobile_visible": true,
      "mobile_icon_url": "https://g.jdycdn.com/shared/images/logo/kms_nav_icon_mobile.png",
      "is_beta": false
    },
    {
      "key": "open_platform",
      "name": "开放平台",
      "url": "https://www.jiandaoyun.com/open",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/open_nav_icon.png",
      "pc_icon_bg": "radial-gradient(111.23% 111.23% at 8.33% 7.08%,rgba(81,105,224,.17) 0,rgba(81,105,224,.1) 50%,rgba(81,105,224,.15) 100%)",
      "mobile_visible": false,
      "is_beta": false
    }
  ],
  "modules": [
    {
      "name": "简道云",
      "key": "dashboard",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "managementUrl": "https://www.jiandaoyun.com/profile",
      "icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_logo.png",
      "suiteId": 1,
      "enable": true,
      "plan_level": "1/trial",
      "plan_name": "试用版",
      "is_beta": false
    },
    {
      "name": "知识库",
      "key": "kms",
      "url": "https://kms.jiandaoyun.com",
      "enable": true,
      "is_beta": false
    },
    {
      "name": "开放平台",
      "key": "open_platform",
      "url": "https://www.jiandaoyun.com/open",
      "enable": true,
      "is_beta": false
    }
  ]
}
```

```

<!-- 接口 -->
https://www.jiandaoyun.com/portal/authn/get_login_user

<!-- 请求参数 -->
{
  "suite_id": 1
}

<!-- 响应参数 -->
{
  "user": {
    "_id": "6a7f3132e6a0aba27cdb3d2b",
    "username": "sys_6a7f3132e6a0aba27cdb3d2a",
    "nickname": "李同学",
    "phone": "15355381414",
    "is_phone_verified": true,
    "email": "",
    "is_email_verified": false,
    "has_password": false,
    "reg_time": "2026-08-14T15:16:02.798Z",
    "login_type": "user",
    "locale": "zh_cn",
    "is_sync": false,
    "owned_tenant_list": [
      {
        "id": "6a7f3132e6a0aba27cdb3d2b",
        "name": "重庆万柯互联网科技有限责任公司"
      }
    ],
    "jsy_migration_type": 0
  },
  "tenant": {
    "id": "6a7f3132e6a0aba27cdb3d2b",
    "corp_id": "6a7f3132e6a0aba27cdb3d2b",
    "integrate_type": "Default",
    "has_link": false,
    "has_name_modify": false,
    "has_crm": false,
    "has_kms": true,
    "name": "重庆万柯互联网科技有限责任公司",
    "owner_id": "6a7f3132e6a0aba27cdb3d2b",
    "wx_enable": false,
    "modules": {
      "new_version": false,
      "new_dash_editor_version": false,
      "aggregate_unlimit": false,
      "corp_theme": false,
      "custom_dashboard": false,
      "relationship": false,
      "automation": false,
      "workflow": false,
      "workflow_migrating": false,
      "report": false,
      "report_retain": false,
      "analytics_beta": false,
      "open_platform_manager": false,
      "screen_recording": false,
      "aggregate_view": false,
      "aggregate_view_open_api_beta": false,
      "jander_beta": false,
      "auto_fill": false,
      "custom_page": false,
      "corp_integrate_switch": false,
      "ai_lab_beta": true,
      "region": false,
      "archive": false,
      "file_manage": false,
      "extended_locale": false,
      "sandbox_app": false,
      "sandbox_app_releasing": false,
      "data_source": false,
      "ai_search_entry_beta": false,
      "app_locale": false,
      "subform_write_exceed": false,
      "governance_beta": false,
      "sales_beta": false,
      "custom_runtime": false,
      "related_form_fill": false,
      "timezone": false,
      "kms": true,
      "bpa": false,
      "app_manage_group": false,
      "private_plugin": false,
      "private_plugin_license": false,
      "member_api_key": true,
      "member_api_key_license": true,
      "pay": false,
      "pay_license": false,
      "corp_metrics": false,
      "corp_executive": false,
      "bpm_delegation": false,
      "ai_lab": true
    },
    "wxwork": {
      "open_data": false
    },
    "size": 1,
    "mfa_enabled": false,
    "is_wxwork_suite_corp": false,
    "watermark": {
      "enabled": false,
      "color": "light",
      "density": "normal"
    },
    "corp_theme": {
      "app_navi_color": "dark"
    },
    "timezone": {
      "label": "Asia/Shanghai",
      "etc": "Etc/GMT-8"
    },
    "region": {
      "use12Hour": false,
      "dateTimeFormat": {
        "ymd_hms": "YYYY-MM-DD HH:mm:ss",
        "ymd_hm": "YYYY-MM-DD HH:mm",
        "ymd": "YYYY-MM-DD",
        "ym": "YYYY-MM",
        "hms": "HH:mm:ss",
        "hm": "HH:mm"
      },
      "weekStart": 1,
      "yearStart": 4,
      "decimalSeparator": ".",
      "thousandSeparator": ","
    },
    "entity_localization": false,
    "ai_capability": {
      "ai_lab": {
        "auto_fill": true,
        "knowledge_qa": true,
        "speech_recognition": true,
        "rich_text_assist": true,
        "field_auto_fill": true,
        "field_suggestion": true,
        "aggregation_formula": true,
        "automation_auto_match": true,
        "llm_node": true,
        "llm_multi_modal_node": true,
        "automation_ai_agent": true,
        "generate_modify_function": true
      }
    }
  },
  "vip": {
    "upgradable_addons": [],
    "vip_buffer_state": "normal",
    "exceed_items": [],
    "pack": {
      "display_level": "trial",
      "display_level_i18n": {
        "zh_cn": "试用版",
        "zh_tw": "試用版",
        "en_us": "Trial",
        "de_de": "Versuch",
        "es_es": "Ensayo",
        "fr_fr": "Procès",
        "id_id": "Uji coba",
        "ja_jp": "トライアル",
        "km_kh": "ការសាកល្បង",
        "ko_kr": "재판",
        "pt_pt": "Julgamento",
        "ru_ru": "Пробный",
        "th_th": "การทดลอง",
        "vi_vn": "Thử nghiệm"
      },
      "level": "trial",
      "service": "basic",
      "users": 30,
      "managers": -1,
      "sys_managers": 1,
      "app": -1,
      "form_data": 300000,
      "data": 750000,
      "excel_import": 5242880,
      "data_xlsx_import": 20971520,
      "data_xls_import": 5242880,
      "file_upload": 128849018880,
      "file_storage": -1,
      "file_max_size": 20971520,
      "file_zip": 1,
      "import_file": 1,
      "data_api": 1,
      "aggregate": 20,
      "form_aggregate": 120,
      "data_backup": 180,
      "logo": 0,
      "user_sync": 1,
      "app_bridge": 1,
      "app_manage_group": 1,
      "permission_management": 1,
      "submit_prompt": 1,
      "print": -1,
      "signature": 1,
      "aggregation": 100,
      "form_aggregation": 1,
      "etl": 50,
      "trigger": 120,
      "trigger_times": 500000,
      "automation_execute_times": 1000000,
      "custom_theme": 1,
      "sso": 1,
      "custom_login": 1,
      "threshold": 1,
      "public_link": 0,
      "dash_style": 1,
      "bpa": 20,
      "open": 1,
      "open_private_plugin": 1,
      "pay": 1,
      "corp_coop": 60,
      "corp_coop_docker": 500,
      "member_freeze": 1,
      "corp_theme": 1,
      "corp_metrics": 1,
      "chat_bot": 1,
      "dash_advanced_charts": 1,
      "watermark": 1,
      "attachment_control": 0,
      "app_attachment_control": 1,
      "data_mask": 1,
      "kms": 1,
      "workbench_advanced": 1,
      "crm": 0,
      "corp_executive": 1,
      "analytics": 1,
      "form_hierarchy_view": 1,
      "analytics_editor": 0,
      "analytics_viewer": 0,
      "automation_loop_node": 1,
      "template_center": 1,
      "official_template": 1,
      "plugin_store": 1,
      "official_plugin": 1,
      "official_data_source": 1,
      "open_application": 1,
      "workflow_simulation": 1,
      "automation_http_node": 1,
      "form_data_page": 1,
      "automation_advanced_node": 1,
      "form_openness_enhancement": 1,
      "ai_compute": 500
    },
    "state": 0,
    "trial_date": "2026-08-28T16:00:00.000Z",
    "expire_date": 0,
    "has_trial": false,
    "has_purchase": true,
    "has_upload_certificate": true,
    "is_buffer": false,
    "is_buffer_expired": false,
    "is_addon_expired": false,
    "is_vip_expired": false,
    "level_expired_days": 10,
    "vip_buf_days_balance": 15
  },
  "platformPlan": {
    "plan": "trial",
    "plan_users": -1,
    "plan_addon": {},
    "plan_gift": {},
    "plan_trial": "2026-08-28T16:00:00.000Z",
    "is_paid": false,
    "is_trial": true,
    "is_free": false,
    "pack": {
      "corp_coop": 60,
      "corp_coop_docker": 500,
      "custom_login": 1,
      "data_api": 1,
      "log_api": 0,
      "sso": 1,
      "corp_executive": 1,
      "user_sync": 1,
      "member_freeze": 1,
      "open": 1,
      "plugin_store": 1,
      "data_source_store": 1,
      "open_private_plugin": 1,
      "open_application": 1,
      "member_api": 1,
      "sys_admin": 5,
      "watermark": 1,
      "corp_theme": 1,
      "ai_lab": 1,
      "excel_import": 5242880
    },
    "upgradable_addons": []
  },
  "member": {
    "member_id": "6a7f3132e6a0aba27cdb3d2b",
    "username": "sys_6a7f3132e6a0aba27cdb3d2a",
    "nickname": "李同学",
    "is_admin": true,
    "is_owner": true,
    "dept": [
      {
        "_id": "6a7f31d6f33abed79701e2dc",
        "dept_no": 1,
        "parent_no": 0,
        "name": "重庆万柯互联网科技有限责任公司",
        "path": [],
        "type": 0,
        "status": 1,
        "seq": 0,
        "departmentId": 1,
        "parentId": 0,
        "order": 0
      }
    ],
    "has_template_center": true,
    "has_file_manage": false,
    "has_help": true,
    "has_chat_bot": true,
    "type": 0,
    "is_open_platform_manager": true,
    "latest_access_suite_id": 1,
    "visible_suite_ids": [
      1
    ]
  },
  "auth_suite_id": 1,
  "suite_modules": [
    {
      "name": "简道云",
      "key": "dashboard",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "managementUrl": "https://www.jiandaoyun.com/profile",
      "icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_logo.png",
      "suiteId": 1,
      "enable": true,
      "plan_level": "1/trial",
      "plan_name": "试用版",
      "is_beta": false,
      "description": "支持低成本快速搭建企业级管理应用，通过功能组合，灵活实现从数据采集、流转到处理、分析的全场景覆盖。",
      "help_doc": "https://hc.jiandaoyun.com/doc/12534",
      "vip_url": "/portal/tenant/6a7f3132e6a0aba27cdb3d2b/admin?_from=1#/vip",
      "guide_img": "https://g.jdycdn.com/shared/images/guide/jdy_guide.png",
      "guide_description": "简道云支持低成本快速搭建企业级管理应用，通过功能组合，灵活实现从数据采集、流转到处理、分析的全场景覆盖。"
    }
  ],
  "navigation_modules": [
    {
      "key": "dashboard",
      "suiteId": 1,
      "name": "工作台",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_nav_icon.png",
      "pc_icon_bg": "linear-gradient(146deg,rgba(59,173,74,.12) 14.15%,rgba(59,173,74,.1) 50.08%,rgba(59,173,74,.12) 93.63%)",
      "mobile_visible": true,
      "mobile_icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_nav_icon_mobile.png",
      "is_beta": false
    },
    {
      "key": "kms",
      "name": "知识库",
      "url": "https://kms.jiandaoyun.com",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/kms_nav_icon.png",
      "pc_icon_bg": "radial-gradient(111.23% 111.23% at 8.33% 7.08%,rgba(250,133,30,.18) 0,rgba(250,133,30,.1) 50%,rgba(250,133,30,.16) 100%)",
      "mobile_visible": true,
      "mobile_icon_url": "https://g.jdycdn.com/shared/images/logo/kms_nav_icon_mobile.png",
      "is_beta": false
    },
    {
      "key": "open_platform",
      "name": "开放平台",
      "url": "https://www.jiandaoyun.com/open",
      "pc_icon_url": "https://g.jdycdn.com/shared/images/logo/open_nav_icon.png",
      "pc_icon_bg": "radial-gradient(111.23% 111.23% at 8.33% 7.08%,rgba(81,105,224,.17) 0,rgba(81,105,224,.1) 50%,rgba(81,105,224,.15) 100%)",
      "mobile_visible": false,
      "is_beta": false
    }
  ],
  "modules": [
    {
      "name": "简道云",
      "key": "dashboard",
      "url": "https://www.jiandaoyun.com/dashboard",
      "mobileUrl": "https://www.jiandaoyun.com/dashboard",
      "managementUrl": "https://www.jiandaoyun.com/profile",
      "icon_url": "https://g.jdycdn.com/shared/images/logo/jdy_logo.png",
      "suiteId": 1,
      "enable": true,
      "plan_level": "1/trial",
      "plan_name": "试用版",
      "is_beta": false
    },
    {
      "name": "知识库",
      "key": "kms",
      "url": "https://kms.jiandaoyun.com",
      "enable": true,
      "plan_level": "103/free",
      "is_beta": false
    },
    {
      "name": "开放平台",
      "key": "open_platform",
      "url": "https://www.jiandaoyun.com/open",
      "enable": true,
      "plan_level": "106/free",
      "is_beta": false
    }
  ]
}

```

```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/tenant/6a7f3132e6a0aba27cdb3d2b/authz/get_permissions
<!-- 请求参数 -->

<!-- 响应参数 -->
{
  "has_team_permission": true,
  "has_role_read_permission": true,
  "has_corp_coop_manage_permission": true,
  "has_dismission_manage_permission": true,
  "is_platform_manager": true
}
```




```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/i18n/get_locale_list
<!-- 请求参数 -->

<!-- 响应参数 -->
[
  {
    "label": "简体中文",
    "value": "zh_cn"
  },
  {
    "label": "繁體中文",
    "value": "zh_tw"
  },
  {
    "label": "English",
    "value": "en_us"
  }
]

```

```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/get_has_unread
<!-- 请求参数 -->
{
  "types": [
    "data_notify",
    "app_log",
    "kms_doc",
    "usage_notify",
    "tenant_coop",
    "open_platform",
    "system_management",
    "operation_notify"
  ],
  "suite_id": 1
}

or

{

}
<!-- 响应参数 -->
{
  "all": true
}

or

{
  "data_notify": false,
  "app_log": true,
  "kms_doc": false,
  "usage_notify": false,
  "tenant_coop": false,
  "open_platform": false,
  "system_management": true,
  "operation_notify": false
}

```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/notice/get
<!-- 请求参数 -->

<!-- 响应参数 -->
{}

```

```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/get_unread_tenant_list
<!-- 请求参数 -->

<!-- 响应参数 -->
[
  {
    "tenant_id": "6a7f3132e6a0aba27cdb3d2b",
    "corp_id": "6a7f3132e6a0aba27cdb3d2b",
    "corp_name": "重庆万柯互联网科技有限责任公司",
    "has_unread": true,
    "is_owner": true
  }
]

```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/suite/unread/list
<!-- 请求参数 -->

<!-- 响应参数 -->
[
  {
    "suite_id": 1,
    "suite_name": "简道云",
    "has_unread": true
  }
]

```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/menu/get
<!-- 请求参数 -->
{
  "suite_id": 1
}
<!-- 响应参数 -->
{
  "suite": [
    {
      "key": "data_notify",
      "label": "数据提醒",
      "children": [
        {
          "key": "comment_mention",
          "label": "评论@我"
        },
        {
          "key": "dash_alert",
          "label": "数据预警"
        },
        {
          "key": "data_export_import",
          "label": "导入导出"
        },
        {
          "key": "crm",
          "label": "CRM"
        }
      ]
    },
    {
      "key": "app_log",
      "label": "应用日志",
      "children": [
        {
          "key": "webhook",
          "label": "数据推送"
        },
        {
          "key": "etl_execute",
          "label": "数据流执行"
        },
        {
          "key": "etl_sync",
          "label": "输出表同步"
        },
        {
          "key": "data_trigger",
          "label": "智能助手执行"
        },
        {
          "key": "app_update",
          "label": "应用变更"
        }
      ]
    },
    {
      "key": "kms_doc",
      "label": "文档动态",
      "children": [
        {
          "key": "kms_doc_mention",
          "label": "提醒查看"
        },
        {
          "key": "kms_coop",
          "label": "文档互动"
        },
        {
          "key": "kms_management",
          "label": "知识库管理"
        }
      ]
    }
  ],
  "platform": [
    {
      "key": "usage_notify",
      "label": "用量提醒"
    },
    {
      "key": "tenant_coop",
      "label": "通讯录管理",
      "children": [
        {
          "key": "corp_coop",
          "label": "互联组织变更"
        },
        {
          "key": "contact",
          "label": "内部组织变更"
        }
      ]
    },
    {
      "key": "open_platform",
      "label": "开放平台"
    },
    {
      "key": "system_management",
      "label": "系统管理",
      "children": [
        {
          "key": "system_export_import",
          "label": "导入导出"
        },
        {
          "key": "message_webhook",
          "label": "消息推送"
        },
        {
          "key": "manage_group",
          "label": "权限变更"
        }
      ]
    },
    {
      "key": "operation_notify",
      "label": "运营通知",
      "children": [
        {
          "key": "product_news",
          "label": "产品动态"
        },
        {
          "key": "app_template_publish",
          "label": "模板发布"
        }
      ]
    }
  ]
}

```


```
<!-- 接口 -->
https://www.jiandaoyun.com/portal/v1/message_center/list
<!-- 请求参数 -->
{
  "skip": 0,
  "limit": 10,
  "type": "data_notify",
  "suite_id": 1
}
<!-- 响应参数 -->
{
  "count": 0,
  "messages": []
}
```
