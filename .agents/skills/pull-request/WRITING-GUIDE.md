# Een pull request schrijven

Schrijf voor een collega die de codebase kent en in GitHub wil beoordelen wat er verandert. Gebruik de EO-schrijfstijl op basis van NEN-ISO 24495-1 over duidelijke taal.

## Titel

- Schrijf in het Nederlands en blijf zo mogelijk binnen 70 tekens.
- Benoem de verandering in de tegenwoordige tijd. Bijvoorbeeld `Help en shellcompletions komen uit dezelfde commanddefinities`.
- Begin met `<KEY> | ` als er een ticket is.

## Beschrijving

1. Begin met een of twee zinnen over wat er verandert en waarom.
2. Voeg context toe als de reviewer die nodig heeft om de verandering te beoordelen.
3. Toon het resultaat als dat helpt. Bij een gewijzigde CLI-aanroep volstaat een kort codeblok en de verwachte uitvoer.

Gebruik actieve zinnen en korte alinea's. Zet drie of meer gelijkwaardige punten in een lijst. Laat tussenkopjes, bestandslijsten en verslagen van geslaagde checks weg.

Plaats bovenaan een Jira-link of een Sentry-link als die bij het werk hoort. Neem het selectievakje letterlijk over en laat het leeg. De auteur vinkt het na de eigen review aan. Haal de templatecommentaren en eventuele toolfooter weg.

## Voorbeeld

**Titel** `PIM-activatie krijgt een eigen commando`

- [ ] _Ik snap elke regel code en heb mijn PR zelf eerst gereviewed. Er zijn voor zover mogelijk linkjes waarmee mijn collega's kunnen testen._

`eo pim` toont welke acties beschikbaar zijn. Contributor activeren vraagt nu om een expliciete aanroep, zodat een verkeerd gespeld subcommando geen activatie start.

```bash
eo pim activate "release voorbereiden"
```
