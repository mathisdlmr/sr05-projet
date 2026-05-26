#import "config.typ": *
#import "layout/page.typ": setup-page
#import "layout/headings.typ": setup-headings
#import "layout/cover.typ": make-cover, make-cover-project
#import "styles.typ": setup-typography
#import "layout/outline.typ": make-outline
#import "features.typ": setup-code
#import "layout/appendix.typ": start-appendix, print-appendix-outline
#import "utils/state.typ": doc-mode

#let project(
    title: "",
    subtitle: none,
    header: none,
    school-logo: none,
    company-logo: none,
    authors: (),
    mentors: (),
    jury: (),
    branch: none,
    semester: none,
    cover-image: none,
    lang: "fr",
    accent-color: default-accent,
    heading-numbering: "1.1",
    features: (),
    defense-date: none,
    footer-text: none,
    company: none,
    date: none,
    body,
) = {
    let lang = load-lang(lang)
    let dict = lang.dict

    set document(author: authors, title: title)

    make-cover-project(
        title: title,
        subtitle: subtitle,
        header: header,
        footer-text: footer-text,
        school-logo: school-logo,
        company-logo: company-logo,
        company: company,
        mentors: mentors,
        jury: jury,
        author: authors,
        branch: branch,
        semester: semester,
        cover-image: cover-image,
        date: date,
        defense-date: defense-date,
        dict: dict,
    )

    show: setup-page.with(features, semester, header, footer-text)
    show: setup-headings.with(features, accent-color, dict, doc-mode)

    show: setup-typography.with(lang)

    show: setup-code.with(features)

    make-outline(dict)

    counter(page).update(0)

    body
}

#let report(
    title: "",
    subtitle: none,
    header: none,
    school-logo: none,
    company-logo: none,
    authors: (),
    mentors: (),
    jury: (),
    branch: none,
    semester: none,
    lang: "fr",
    accent-color: default-accent,
    heading-numbering: "1.1",
    cover-image: none,
    features: (),
    defense-date: none,
    footer-text: none,
    company: none,
    date: none,
    body,
) = {
    let lang = load-lang(lang)
    let dict = lang.dict

    set document(author: authors, title: title)

    make-cover(
        title: title,
        subtitle: subtitle,
        header: header,
        footer-text: footer-text,
        school-logo: school-logo,
        company-logo: company-logo,
        company: company,
        mentors: mentors,
        cover-image: cover-image,
        jury: jury,
        author: authors.first(),
        branch: branch,
        semester: semester,
        date: date,
        defense-date: defense-date,
        dict: dict,
    )

    show: setup-page.with(features, semester, header, footer-text)
    show: setup-headings.with(features, accent-color, dict, doc-mode)

    show: setup-typography.with(lang)

    show: setup-code.with(features)

    make-outline(dict)

    counter(page).update(0)

    body
}