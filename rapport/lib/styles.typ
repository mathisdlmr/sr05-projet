#let setup-typography(lang, body) = {
    set text(
        lang: lang.code,
        //font: "Libertinus Serif",   // example
        size: 13pt,
    )

    set par(
        first-line-indent: 1.5em,
        justify: true,
    )

    // In french, first paragraph is also indented
    // This should be default behavior, but meanwhile...
    set par(first-line-indent: (amount: 1.5em, all: true)) if lang.code == "fr"

    // Headings should NOT be justified or indented
    show heading: it => {
        set par(first-line-indent: 0em, justify: false)
        it
    }

    // Code blocks
    show raw: set text(
        size: 10pt,
    )

    // Figure captions in italic
    show figure.caption: emph

    // underline 'external' links
    // This should be a separate function probably, like external-link()
    show link: it => {
        if type(it.dest) == str {
            underline(it)
        } else {
            it
        }
    }

    body
}
