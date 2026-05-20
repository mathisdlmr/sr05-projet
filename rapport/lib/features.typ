#let setup-code(features, body) = {
    import "@preview/codly:1.3.0": *
    show: codly-init.with()

    import "@preview/codly-languages:0.1.10": *
    codly(languages: codly-languages)
    body
}
