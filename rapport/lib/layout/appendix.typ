#import "../utils/state.typ": doc-mode
#import "../config.typ": load-lang

#let print-appendix-outline(dict) = {
    pagebreak()
    heading(supplement: [extra], numbering: none)[#dict.appendix_outline_title]
    outline(target: heading.where(supplement: [Appendix]), title: none)
}

#let start-appendix(body) = {
    doc-mode.update("appendix")

    set heading(numbering: "A", supplement: [Appendix])

    counter(heading).update(0)

    print-appendix-outline(load-lang("fr").dict)

    body
}
