#let supported-langs = ("en", "fr", "ar")

#let IMAGE_BOX_MAX_WIDTH = 120pt
#let IMAGE_BOX_MAX_HEIGHT = 50pt

#let default-accent = (252, 210, 22)

#let load-lang(lang) = {
    if not supported-langs.contains(lang) {
        panic("Unsupported language: " + lang)
    }
    (
        code: lang,
        dict: json("resources/i18n/" + lang + ".json"),
    )
}
