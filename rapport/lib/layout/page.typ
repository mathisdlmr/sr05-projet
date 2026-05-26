#let setup-page(features, semester, header-text, footer-text, body) = {
    import "@preview/hydra:0.6.2": hydra

    set page(header: context {
        if hydra(1) != none {
            align(end)[#hydra(1)]
            line(length: 100%)
        }
        // reset footnote counter in header to start at 1 on each new page
        counter(footnote).update(0)
    })

    set page(
        numbering: none,
        number-align: center,
        footer: context {
            // Omit page number on the first page
            let page-number = counter(page).get().at(0)

            if page-number > 0 {
                line(length: 100%, stroke: 0.5pt)
                v(-2pt)
                grid(
                    columns: (40%, 20%, 40%),
                    align: (left, center, right),
                    footer-text, counter(page).display(), semester,
                )
            }
        },
    )

    body
}
