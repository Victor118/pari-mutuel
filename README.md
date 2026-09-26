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
| `Money` | value object | `Money` | par valeur |

**`Amount` en `int64` de centimes, jamais en flottant.** `0.1` n'est pas représentable
en binaire ; les erreurs s'accumulent et l'invariante ci-dessus cesse de tenir. Un
entier dans la plus petite unité indivisible est exact par construction. `int64` plafonne
à ~92 000 milliards d'euros, ce qui laisse de la marge.

**`Amount` est un `struct` à champ non exporté**, pas un `type Amount int64`. Un type
dérivé se construit par conversion (`Amount(-500)` compile, y compris hors du package),
donc aucune invariante ne tiendrait. Le struct rend le constructeur incontournable, tout
en restant comparable par `==` et utilisable comme clé de map. Sa valeur zéro vaut
zéro centime : valide et pleine de sens.

**Les champs de `Pool` ne sont pas exportés.** Un champ public rend l'invariante
contournable après construction : `p.Resolver = "attaquant"` suffisait à prendre le
contrôle de la résolution, `p.ClosesAt` à rouvrir un pool fermé, et `p.Outcomes`, étant
une slice, à ajouter une issue en contournant le contrôle d'unicité. Les tests vivant en
`package pool_test`, aucun n'y accédait — le passage en minuscule n'a pas coûté une ligne
de test. Les accesseurs viendront quand un appelant les réclamera, et `Outcomes()` devra
cloner à la sortie comme `NewPool` clone à l'entrée : une slice s'encapsule aux deux bouts.

**`Money` porte la devise, `Amount` non.** `Amount` est une quantité sans unité ;
`Money` est le couple montant + devise. Un pool n'acceptant qu'une seule devise,
la faire descendre dans `Amount` la répéterait sur chaque entrée de
`stakedByOutcome`, dupliquant un fait que le pool porte déjà — la faute de
`PoolState` sous un autre nom. La séparation préserve trois choses : la valeur zéro
d'`Amount` reste utile, donc la map peut rester creuse et une issue sans mise rend
« 0 » dans la devise du pool ; `Add` reste de l'arithmétique pure, sans retour
d'erreur, puisque sans devise il n'y a pas de mélange possible ; et le type le plus
testé du domaine n'a pas bougé.

La règle qui en découle : **`Money` à la surface, `Amount` à l'intérieur.** La devise
n'apparaît que là où des montants hétérogènes pourraient se rencontrer, c'est-à-dire
aux frontières de l'agrégat — `PlaceBet` en entrée, `TotalBetOnOutcome` en sortie.

**La garde vit dans l'agrégat**, pas à la frontière. `PlaceBet` compare la devise de
la mise à celle du pool et refuse. Déléguer cette vérification à la couche appelante
aurait laissé l'invariante contournable depuis l'extérieur, exactement le trou fermé
en passant les champs de `Pool` en privé.

`Currency` reste un `type Currency string` nu, contrairement à `Amount`. Ce n'est pas
un objet valeur porteur d'arithmétique mais un identifiant opaque : sur une chaîne,
une dénomination est exacte et sensible à la casse (`uatom`, `ibc/27394F…`).
Normaliser la casse la casserait. `NewPool` refuse donc une devise vide ou blanche
sans jamais la réécrire.

## Cycle de vie d'un pool

Quatre états, dont un seul est stocké :

| état | source | terminal |
|---|---|---|
| ouvert | `now <= closesAt` | non |
| clos | `now > closesAt` | non |
| résolu | `winner != ""` | oui |
| annulé | champ `cancelled` | oui |

**Ce qui se déduit ne se stocke pas.** Un champ `State` avait d'abord été écrit, puis
supprimé : il dupliquait une information déjà portée par `closesAt` et par `winner`, sans
qu'aucune contrainte ne garantisse leur cohérence — un pool résolu pouvait annoncer
`"open"`. Deux représentations de la même question finissent toujours par diverger. Étant
de surcroît exporté, il permettait de réautoriser les mises sur un pool résolu.

`cancelled` est le seul état qui ne se déduit de rien, ni du temps ni du gagnant. C'est à
ce titre qu'il mérite un champ.

**Résolu et annulé s'excluent dans les deux sens.** Sans cette exclusion, un pool pourrait
autoriser simultanément le paiement des gains et le remboursement des mises sur le même
pot. La résolution est définitive, y compris pour le resolver légitime : une fois des gains
réclamés, réécrire le gagnant rendrait ces paiements incohérents. Une éventuelle fenêtre de
correction se modélisera comme une transition explicite et datée, pas comme un assouplissement.

L'annulation, elle, reste idempotente. Sans charge utile à écraser, une seconde annulation
réaffirme le même état, là où une seconde résolution aurait pu changer le gagnant.

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
- [x] `Amount` — construction, addition, `Zero`
- [x] `NewAmount` refuse un montant négatif ; bornes testées de `MinInt64` à `MaxInt64`
- [x] placer une mise, refus d'une issue inconnue
- [x] accumulation de deux mises sur la même issue
- [x] consulter le total misé sur une issue
- [x] refus d'une mise après `ClosesAt`
- [x] résolution par le resolver désigné : définitive, sur une issue existante
- [x] annulation par le resolver désigné, exclusive de la résolution
- [x] refus d'une mise sur un pool résolu ou annulé
- [x] champs de `Pool` non exportés
- [x] `Money` et `Currency` : un pari n'est accepté que dans la devise du pool
- [x] `NewPool` exige une devise non vide

Liste de tests en cours :

- [ ] refus d'une mise de zéro (règle du pari, pas de la monnaie)
- [ ] mises par compte — `PlaceBet` reçoit `account` sans le stocker, donc aucune trace
      de qui a misé quoi ; ni gain ni remboursement n'est calculable en l'état
- [ ] règlement du pool : `gain = mise × masse totale / masse gagnante`
- [ ] politique du reste (breakage) — la somme des gains doit boucler au centime
- [ ] remboursement des mises après annulation

Dettes connues, assumées :

- `Amount.Add` panique sur dépassement de capacité au lieu de rendre une erreur. Deux
  montants valides peuvent sommer au-delà de `MaxInt64`, ce n'est donc pas un état
  impossible : c'est un arbitrage sur la magnitude du domaine, où un crash vaut mieux
  qu'un solde négatif silencieux.
- La multiplication du règlement débordera bien avant l'addition. `mise × masse totale`
  atteint 10¹⁸ centimes avec un pot de dix millions d'euros, à un facteur 9 de `MaxInt64`.
  Il faudra un intermédiaire 128 bits ou `math/big`, et là le dépassement sera un cas
  métier atteignable, pas une assertion.
- Le message de `panic` d'`Amount.Add` est le seul texte français du code, tous les
  autres messages étant en anglais.
- `creator` et `question` sont écrits et jamais lus. Go ne signale pas les champs de
  struct inutilisés, et aucun accesseur n'existe encore.
