# pari-mutuel

Un projet d'apprentissage : mettre en œuvre le **TDD en Go** sur un domaine métier
qui ne pardonne pas l'à-peu-près — le **pari mutuel**.

Le code n'est pas le but. La façon dont il apparaît, si.

## Pourquoi le pari mutuel

Contrairement aux cotes fixes, le pari mutuel redistribue l'intégralité des mises
entre les gagnants. Ça impose une invariante dure :

```
somme des gains distribués + commission + breakage == masse totale des mises
```

Une égalité **exacte**, au centime, sur des milliers d'opérations. C'est un domaine
où les raccourcis se voient : arithmétique flottante, arrondis implicites, division
entière dont on ignore le reste. Parfait pour un exercice où les tests doivent
réellement contraindre le code.

## Discipline appliquée

- **Red / Green / Refactor**, dans cet ordre, sans exception.
- **Un seul rouge à la fois.** Toute idée qui surgit en cours de cycle part sur la
  liste de tests (plus bas) au lieu d'ouvrir un second chantier.
- **Voir le test échouer** avant d'écrire l'implémentation. Un test jamais passé au
  rouge ne prouve rien — il peut très bien ne rien tester.
- **Le minimum pour passer au vert.** La validation, les cas limites et les erreurs
  métier sont des cycles à part entière, pas des ajouts opportunistes.
- **YAGNI vaut aussi pour les tests.** Pas de test sur une méthode qu'aucun appelant
  ne réclame encore.

## Conventions de test

**Package externe.** Les tests vivent dans `package pool_test`, pas `package pool` —
même dossier, package distinct (Go autorise cette exception). Conséquence : ils ne
voient que l'API exportée, exactement comme un vrai client du package.

Ce n'est pas du purisme. Ça force deux choses :

1. concevoir l'API avant l'implémentation, puisque le test ne peut rien atteindre d'autre ;
2. pouvoir réécrire tout l'intérieur d'un type sans toucher une ligne de test — sinon
   l'étape *refactor* devient impraticable et les tests passent de filet à frein.

Un test qui doit vraiment atteindre un interne irait dans un `*_internal_test.go`
séparé, en `package pool`. Il n'y en a aucun pour l'instant.

**Stdlib uniquement.** Pas de `testify`, pas d'assertion library. `t.Fatalf` joue le
rôle de `require` (arrêt immédiat), `t.Errorf` celui de `assert` (on continue). Les
comparaisons passent par `==`, vérifié à la compilation — ce qu'une lib à base de
`interface{}` et de réflexion ne garantit pas.

## Modèle

| Type | Nature | Récepteur | Identité |
|---|---|---|---|
| `Pool` | entité | `*Pool` | par `PoolID` |
| `Amount` | value object | `Amount` | par valeur |
| `Question` | value object | `Question` | par valeur |

**`Amount` en `int64` de centimes, jamais en flottant.** `0.1` n'est pas représentable
en binaire ; les erreurs s'accumulent et l'invariante ci-dessus cesse de tenir. Un
entier dans la plus petite unité indivisible est exact par construction. `int64` plafonne
à ~92 000 milliards d'euros, ce qui laisse de la marge.

**`Amount` est un `struct` à champ non exporté**, pas un `type Amount int64`. Un type
dérivé se construit par conversion (`Amount(-500)` compile, y compris hors du package),
donc aucune invariante ne tiendrait. Le struct rend le constructeur incontournable, tout
en restant comparable par `==` et utilisable comme clé de map. Sa valeur zéro vaut
zéro centime : valide et pleine de sens.

## Lancer les tests

Le projet cible Go 1.24 et fournit un devcontainer.

```bash
go test ./...
go test -v -run TestPlaceBet ./pool/
go test -cover ./...
```

## Où on en est

Implémenté :

- [x] création d'un pool (≥ 2 issues, issues uniques)
- [x] `Amount` — construction et addition
- [x] placer une mise, refus d'une issue inconnue
- [x] consulter le total misé sur une issue

Liste de tests en cours :

- [ ] accumulation de deux mises sur la même issue
- [ ] `NewAmount` refuse un montant négatif
- [ ] refus d'une mise sur un pool non ouvert, ou après `ClosesAt`
- [ ] refus d'une mise de zéro (règle du pari, pas de la monnaie)
- [ ] règlement du pool : `gain = mise × masse totale / masse gagnante`
- [ ] politique du reste (breakage) — la somme des gains doit boucler au centime

Dettes connues, assumées :

- Les champs de `Pool` sont exportés, donc ses invariantes sont contournables après
  construction (`p.State = Settled`).
