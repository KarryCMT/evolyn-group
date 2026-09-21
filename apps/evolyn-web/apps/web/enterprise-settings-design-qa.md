# Design QA: 管理后台·企业设置

## Evidence

- Reference screenshots: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-75673056-1d0c-4e59-b2ff-3dc6bc2d5d2f.png` through `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-4e31ea2f-0838-4815-969f-f2cd6751eebb.png`
- Browser-rendered implementation: `http://127.0.0.1:11000/tenant/enterprise-settings`
- QA viewport: 1280 × 720, light theme

## Comparison checklist

- [x] Overall composition matches the reference: a white settings surface with three vertically stacked groups and generous horizontal spacing.
- [x] Section hierarchy, teal title marker, row labels, descriptions, dividers, switches, buttons, links and the language selector match the reference hierarchy and visual weight.
- [x] Enterprise security, enterprise culture and enterprise collaboration content is present in the same order as the screenshots.
- [x] SSO configuration covers SAML 2.0, custom interface and CAS variants, including validation and secret generation.
- [x] Enterprise theme color popover matches the compact swatch-based interaction and persists the selected accent color in the current session.
- [x] Custom login style opens a full-screen preview/editor with brand preview, artwork, login method controls and a fixed footer.
- [x] Reminder suppression opens a member/application mapping dialog with add and remove row interactions.
- [x] Browser console reported no runtime errors while exercising the implemented interactions.
- [x] Keyboard-focusable native controls and explicit accessible labels are present for switches, theme swatches and dialog actions.

## Intentional differences

- Product branding and sample content use **灵衍云** to comply with repository branding rules.
- The custom-login illustration reuses the project's existing login asset instead of copying the external screenshot artwork.
- Placeholder help/configuration actions show explicit feedback until their backend services are connected.

## Result

**Passed.** The implementation is visually aligned with the supplied desktop references at the tested viewport, key dialogs and state transitions were exercised, and no blocking visual or interaction defects remain.
