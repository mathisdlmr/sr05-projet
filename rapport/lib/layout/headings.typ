#let setup-headings(features, accent-color, dict, doc-mode, body) = {

    set heading(numbering: "1.1", supplement: dict.chapter)

    show heading.where(supplement: [#dict.chapter]): it => {
        if it.level == 1 and it.numbering != none {
            if features.contains("full-page-chapter-title") {
                pagebreak()
                set page(footer: none)

                block()[
                    #v(1fr)
                    #text(size: 30pt)[#it.supplement #counter(heading).display()]
                    #linebreak()
                    #text(weight: "bold", size: 36pt)[#it.body]
                    #line(start: (0%, -1%), end: (15%, -1%), stroke: 2pt + accent-color)
                    #v(1fr)
                ]
                pagebreak()
            } else {
                pagebreak()
                text(size: 30pt)[#it.body]
            }
        } else {
            if it.level == 1 {
                pagebreak()
                block()[
                    #set text(size: 30pt)
                    #v(5pt)
                    #it
                    #v(20pt)
                ]
            }
            else {
                block(below: 20pt, above: 20pt)[#it]
            }
        }
    }

    // appendix titles
    show heading.where(supplement: [Appendix]): it => {
        pagebreak()
        [#dict.appendix #counter(heading).display() : #it.body]
    }

    //show ref.where(supplement: [Appendix]): it => it.supplement(none)
    show ref: it => {
        let el = it.element

        if el == none { return it }
        if el.func() != heading { return it }
        if el.supplement != [Appendix] { return it }

        let new-supp = [Annexe]  // ← mets ce que tu veux ici
          // Numéro du heading (A, B, C…)
        let nums = counter(heading).at(el.location())
        let num = numbering(el.numbering, ..nums)

        link(el.location(), [#new-supp #num])
    }

    body
}
