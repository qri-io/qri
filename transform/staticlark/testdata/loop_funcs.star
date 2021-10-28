load("math.star", "math")


def stddev(ls):
  total = 0
  for x in ls:
    total += x
  n = len(ls)
  mean = total / n
  result = 0
  for x in ls:
    diff = x - mean
    result += diff * diff
  variance = result / n
  return math.sqrt(variance)


def gcd_debug(a, b):
  print("gcd starting")
  for n in range(20):
    print("gcd a = %d, b = %d", a, b)
    if a == b:
      print("gcd break at step %d", n)
      break
    else:
      print("still going")
    if a > b:
      a = a - b
    else:
      b = b - a
  print("gcd returns %d", a)
  return a
