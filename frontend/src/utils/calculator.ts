/**
 * 计算器工具函数
 */

/**
 * 加法
 */
export function add(a: number, b: number): number {
  return a + b;
}

/**
 * 减法
 */
export function subtract(a: number, b: number): number {
  return a - b;
}

/**
 * 乘法
 */
export function multiply(a: number, b: number): number {
  return a * b;
}

/**
 * 除法
 */
export function divide(a: number, b: number): number {
  if (b === 0) {
    throw new Error('除数不能为零');
  }
  return a / b;
}
