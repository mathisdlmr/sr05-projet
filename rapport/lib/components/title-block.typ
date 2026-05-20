#let title-block(title, subtitle, date) = {
    align(center + horizon)[
        #if subtitle != none {
            text(size: 14pt, tracking: 2pt)[
                #smallcaps[
                #subtitle
                ]
            ]
        }
        #line(length: 100%, stroke: 0.5pt)
        #text(size: 18pt, weight: "bold")[#title]
        #line(length: 100%, stroke: 0.5pt)
    ]

    align(center)[#date]
}