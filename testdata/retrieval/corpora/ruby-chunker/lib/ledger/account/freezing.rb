# frozen_string_literal: true

module Ledger
  class Account
    def freeze!(reason)
      @frozen_reason = reason
      @frozen_at = Time.now
      self
    end

    def unfreeze!
      @frozen_reason = nil
      @frozen_at = nil
      self
    end

    def frozen_account?
      !@frozen_at.nil?
    end
  end
end
