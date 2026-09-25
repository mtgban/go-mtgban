#!/usr/bin/env python3
"""Grade one game's Cardmarket census, as census.sh leaves it.

    grade.py GAME [LABEL] [TOP]      report on walk-LABEL.tsv (default master)
    grade.py GAME --diff BASE HEAD   every product whose landing differs

The report prints: verdict counts; each landing graded against its anchor
(the datastore's own Cardmarket id, else CardTrader's link); anchor
disagreements, and foreign, refused, twin and error value, by shelf;
printings two products price; landings CardTrader's program names another
row for; and price outliers against the published TCGplayer market.
Reads $OUT/GAME (default ~/src/claude-scratchpad/cardmarket-census/GAME).
"""
import csv
import json
import os
import re
import sys
from collections import Counter, defaultdict

game = sys.argv[1]
D = os.path.join(os.environ.get('OUT', os.path.expanduser('~/src/claude-scratchpad/cardmarket-census')), game)


def walk(label):
    with open(f'{D}/walk-{label}.tsv') as f:
        return {r['product_id']: r for r in csv.DictReader(f, delimiter='\t')}


if sys.argv[2:3] == ['--diff']:
    base, head = walk(sys.argv[3]), walk(sys.argv[4])
    key = lambda r: (r['verdict'], r['card_id'], r['card_id_foil'], r['err'])
    moved = [k for k in base if k in head and key(base[k]) != key(head[k])]
    print(f'{len(moved)} of {len(base)} products change')
    print(Counter((base[k]['verdict'], head[k]['verdict']) for k in moved))
    for k in moved:
        a, b = base[k], head[k]
        print(f"{k}\t{a['exp_name']}\t{a['name']}\t{key(a)} -> {key(b)}")
    sys.exit()

label = sys.argv[2] if len(sys.argv) > 2 else 'master'
TOP = int(sys.argv[3]) if len(sys.argv) > 3 else 12
rows = walk(label)
be = json.load(open(f'{D}/backend.json'))
U = {u['UUID']: u for u in be['uuids']}
tcg2u = be['ext'].get('tcgplayer', {})
cm2u = be['ext'].get('cardmarket', {})
bridge = {int(k): v for k, v in json.load(open(f'{D}/data/{game}-bridge.json')).items()}
guide = {g['idProduct']: g for g in json.load(open(f'{D}/data/{game}-guide.json'))}
# The label check reads the datastores publishing each card under its uuid
# as `id`; Magic's AllPrintings (a gigabyte, skipped unread), Lorcana's and
# Riftbound's upstream shapes do not, and get no label check.
data = {} if game == 'magic' else json.load(open(f'{D}/data/{game}-datastore.json'))['data']
raw = [c for c in data.get('cards') or [] if isinstance(c.get('id'), str) and c.get('name')]
byid = {c['id']: c for c in raw}
bycm = defaultdict(list)
for b in json.load(open(f'{D}/data/{game}-blueprints.json')):
    for i in b.get('card_market_ids') or []:
        bycm[i].append(b)
try:
    tcg = json.load(open(f'{D}/dumps/TCGMarket.json'))['inventory']
except FileNotFoundError:
    tcg = {}


def tr(k):
    return guide.get(int(k), {}).get('trend') or 0


def card(u):
    x = U.get(u)
    return (x['Set'], x['Number'], x['Name']) if x else None


def anchor(k):
    t = bridge.get(int(k))
    return cm2u.get(k), (tcg2u.get(str(t)) if t else None), t


print(f'== {game} {label}: {len(rows)} products', dict(Counter(r['verdict'] for r in rows.values())))
cls = Counter()
dis = defaultdict(list)
for k, r in rows.items():
    own, tu, t = anchor(k)
    if r['verdict'] != 'landed':
        cls[r['verdict'] + '-' + ('own-id' if own else 'bridge-knows' if tu else 'bridge-tcg-unknown' if t else 'no-anchor')] += 1
        continue
    a = own or tu
    if not a:
        cls['bridge-tcg-unknown' if t else 'no-anchor'] += 1
    elif card(a) == card(r['card_id']):
        cls[('own-id' if own else 'bridge') + '-agree'] += 1
    else:
        cls[('own-id' if own else 'bridge') + '-DISAGREE'] += 1
        dis[r['exp_name']].append((tr(k), k, r['name'][:40], r['number'], card(r['card_id']), card(a)))
print(dict(cls.most_common()))

print('\n== anchor disagreements by shelf')
for e, l in sorted(dis.items(), key=lambda x: -sum(y[0] for y in x[1]))[:TOP]:
    x = max(l)
    print(f'  {len(l):4} €{sum(y[0] for y in l):9.2f} {e[:40]:40} top {x[1]} {x[2]} #{x[3]} walk {x[4]} anchor {x[5]}')

for v in ('foreign', 'refused', 'twin', 'error'):
    by = defaultdict(lambda: [0, 0.0, None, 0])
    for k, r in rows.items():
        if r['verdict'] != v:
            continue
        e = by[r['exp_name']]
        e[0] += 1
        e[1] += tr(k)
        e[3] += bool(anchor(k)[1])
        if not e[2] or tr(k) > tr(e[2]):
            e[2] = k
    if not by:
        continue
    print(f'\n== {v}: {sum(x[0] for x in by.values())} products, €{sum(x[1] for x in by.values()):.0f}')
    for en, x in sorted(by.items(), key=lambda y: -y[1][1])[:TOP]:
        r = rows[x[2]]
        print(f"  {x[0]:4} €{x[1]:9.2f} bridged {x[3]:3} {en[:36]:36} top {x[2]} {r['name'][:36]} #{r['number']} €{tr(x[2])} {r['err'][:28]}")

pricing = defaultdict(set)
for k, r in rows.items():
    if r['verdict'] == 'landed':
        pricing[r['card_id']].add(k)
multi = [(u, sorted(p)) for u, p in pricing.items() if len(p) > 1]
print(f'\n== printings priced by more than one product: {len(multi)}')
for u, ps in sorted(multi, key=lambda x: -max(tr(p) for p in x[1]))[:TOP]:
    print('  ', u, card(u), [(p, rows[p]['exp_name'][:22], rows[p]['name'][:34], rows[p]['number'], tr(p)) for p in ps])

STOP = {'promo', 'card', 'set', 'version', 'edition', 'the', 'a', 'of', 'and', 'foil', 'normal', 'non'}


def toks(s):
    return {t for t in re.split(r'[^a-z0-9]+', (s or '').lower()) if t and t not in STOP and len(t) > 1}


bynum = defaultdict(list)
for c in raw:
    bynum[(c.get('number', ''), c['name'])].append(c)
contra = []
for k, r in rows.items():
    bl = bycm.get(int(k), [])
    c = byid.get(r['card_id'])
    if r['verdict'] != 'landed' or not bl or not c:
        continue
    ct = toks(bl[0].get('version'))
    if not ct or ct <= toks(c.get('variant')):
        continue
    sib = [s for s in bynum[(c.get('number', ''), c['name'])] if s['id'] != c['id'] and ct <= toks(s.get('variant'))]
    if sib:
        contra.append((tr(k), k, r['exp_name'][:24], r['name'][:34], bl[0]['version'], c['id'], c.get('variant'), [(s['id'], s.get('variant')) for s in sib][:2]))
print(f'\n== landings whose CardTrader program names another row: {len(contra)}')
for x in sorted(contra, reverse=True)[:TOP]:
    print('  ', x)

out = []
for k, r in rows.items():
    t = [e['price'] for e in tcg.get(r['card_id'], [])]
    m = tr(k) * 1.14
    if r['verdict'] != 'landed' or not t or m < 20 or t[0] <= 0:
        continue
    ratio = max(m, t[0]) / min(m, t[0])
    if ratio >= 4:
        out.append((round(abs(m - t[0]), 2), round(ratio, 1), k, r['exp_name'][:26], r['name'][:36], r['number'], r['card_id'], 'bridge' if int(k) in bridge else 'name', round(m, 2), t[0]))
print(f'\n== landed prices 4x off TCGplayer market (Cardmarket at least $20): {len(out)}')
for x in sorted(out, reverse=True)[:TOP]:
    print('  ', x)
