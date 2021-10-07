def func_a():
  func_b()
  func_d()
  func_g()

def func_b():
  func_d()

def func_c():
  func_e()

def func_d():
  print('D')

def func_e():
  print('E')

def func_f():
  print('F')

def func_g():
  print('G')

def top_level_func():
  func_a()

top_level_func()
