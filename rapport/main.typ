#import "lib/lib.typ": project, start-appendix

#show: project.with(
    title: "Projet de SR05 (Systèmes et Algorithmes Répartis)",
    subtitle: "Le jeu du loup-garou, mais distribué",
    authors: (
        "Mathis DELMAERE",
        "Eliott THOMANN",
        "Elias LORGNIER",
        "Éric BJÄRSTÅL",
    ),
    // jury: ( "Pr. John Smith", "Pr. Jane Doe" ),
    semester: "P26",
    footer-text: "UV - Sujet", // Text used in left side of the footer
    cover-image: image("assets/loup-garou.png", width: 50%),
    // header: [ Université de Technologie de Compiègne ],
    // defense-date: "September 10th, 2025", // OPTIONAL: Needs the jury list to be displayed for the defense date to be added

    lang: "fr",  // optional, default is "fr"
    features: ("header-chapter-name"), // All features are optional and not activated by default. Include the desire features.
    accent-color: rgb(251, 223, 75), // OPTIONAL: Change the default accent color of the document
)

#import "@preview/glossarium:0.5.9": gls, glspl, make-glossary, print-glossary, register-glossary
#import "abbreviations.typ": abbreviations-entry-list
#import "glossary.typ": glossary-entry-list
#register-glossary(abbreviations-entry-list)
#register-glossary(glossary-entry-list)
#show: make-glossary

#include "chapters/01_introduction.typ"
#include "chapters/02_developpement.typ"
#include "chapters/03_conclusion.typ"

//#bibliography("bibliography.bib")

//#show: start-appendix

//#include "chapters/11_annexe1.typ"
